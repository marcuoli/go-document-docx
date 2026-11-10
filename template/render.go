// Package template renders caller-supplied DOCX templates and composes closing
// pages. Templates are literal placeholders, not executable Python/Jinja programs.
package template

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/internal/nativecontent"
	"github.com/marcuoli/go-document-docx/internal/opc"
	"github.com/mmonterroca/docxgo/v2/domain"
	upstream "github.com/mmonterroca/docxgo/v2/pkg/template"
)

// Image contains caller-provided image bytes and optional dimensions in twips.
// Image placeholders must occupy an entire paragraph, including inside a cell/header.
type Image struct {
	Data          []byte
	Width, Height int
}

// Options configures new content and caller-supplied image placeholders. It never
// looks up customer identities, assets, translation services or installed fonts.
type Options struct {
	Document document.Options
	Images   map[string]Image
}

// Render substitutes strict {{key}} text placeholders, including split runs and
// whitespace around keys. Values stay literal and are substituted once. Dotted
// keys address literal map keys, not object properties. Unsupported Jinja controls,
// field-adjacent/unrepresented text locations and missing keys fail before output.
func Render(ctx context.Context, input []byte, values map[string]string, options Options) ([]byte, error) {
	editor, err := editing.Open(ctx, input, options.Document)
	if err != nil {
		return nil, err
	}
	pkg, err := opc.Read(ctx, input)
	if err != nil {
		return nil, err
	}
	syntax := map[string]string{}
	counts := map[string]int{}
	occurrences := 0
	for name := range pkg.Parts {
		if !strings.HasPrefix(name, "word/") || !strings.HasSuffix(name, ".xml") {
			continue
		}
		xml, err := pkg.XML(name)
		if err != nil {
			return nil, err
		}
		var excluded strings.Builder
		for _, node := range all(xml.Root()) {
			for _, attribute := range node.Attr {
				if strings.Contains(attribute.Value, "{{") || strings.Contains(attribute.Value, "{%") {
					return nil, fmt.Errorf("template placeholders in XML attributes are unsupported")
				}
			}
			if node.NamespaceURI() == opc.Word && (node.Tag == "instrText" || node.Tag == "delText") {
				excluded.WriteString(node.Text())
				if strings.Contains(node.Text(), "{{") || strings.Contains(node.Text(), "{%") {
					return nil, fmt.Errorf("template placeholder is in unsupported field/revision text")
				}
			}
		}
		if strings.Contains(excluded.String(), "{{") || strings.Contains(excluded.String(), "{%") {
			return nil, fmt.Errorf("fragmented template syntax in field/revision text is unsupported")
		}
		for _, paragraph := range opc.Elements(xml.Root(), opc.Word, "p") {
			text := paragraphText(paragraph)
			matches, err := placeholders(text)
			if err != nil {
				return nil, err
			}
			for original, key := range matches {
				syntax[original] = key
			}
			for _, match := range placeholderPattern.FindAllStringSubmatch(text, 4097) {
				counts[match[0]]++
				occurrences++
				if occurrences > 4096 {
					return nil, fmt.Errorf("template exceeds 4096 placeholders")
				}
			}
		}
	}
	data := map[string]string{}
	total := 0
	for key, value := range values {
		if !keyPattern.MatchString(key) {
			return nil, fmt.Errorf("invalid template key")
		}
		if err := opc.Text(value); err != nil {
			return nil, err
		}
		if len(value) > 64<<20-total {
			return nil, fmt.Errorf("template values exceed 64 MiB")
		}
		total += len(value)
		data[key] = value
	}
	imageParagraphs := map[string][]domain.Paragraph{}
	if err := nativecontent.Walk(ctx, editor.Native(), func(p domain.Paragraph) error {
		matches, err := placeholders(p.Text())
		if err != nil {
			return err
		}
		for original, key := range matches {
			if _, isImage := options.Images[key]; isImage {
				if strings.TrimSpace(p.Text()) != original {
					return fmt.Errorf("image placeholder must occupy an entire paragraph")
				}
				imageParagraphs[key] = append(imageParagraphs[key], p)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	for key := range options.Images {
		if !keyPattern.MatchString(key) {
			return nil, fmt.Errorf("invalid image template key")
		}
		if _, duplicate := data[key]; duplicate {
			return nil, fmt.Errorf("template key is both text and image")
		}
		if len(imageParagraphs[key]) == 0 {
			return nil, fmt.Errorf("image placeholder %q not found", key)
		}
		data[key] = ""
	}
	originals := make([]string, 0, len(syntax))
	for original, key := range syntax {
		if _, ok := data[key]; !ok {
			return nil, fmt.Errorf("missing template key %q", key)
		}
		originals = append(originals, original)
	}
	slices.Sort(originals)
	// Literal ReplaceText preserves fields that the upstream MergeTemplate API
	// may clear. Its skipped-layout receipt is enforced rather than losing fields.
	// Opaque intermediate tokens prevent a value containing another placeholder
	// from being evaluated by subsequent replacements.
	random := [16]byte{}
	if _, err := rand.Read(random[:]); err != nil {
		return nil, err
	}
	prefix := "DOCUMENT_VALUE_" + hex.EncodeToString(random[:]) + "_"
	for _, value := range data {
		if strings.Contains(value, prefix) {
			return nil, fmt.Errorf("template token collision")
		}
	}
	for _, part := range pkg.Parts {
		if bytes.Contains(part, []byte(prefix)) {
			return nil, fmt.Errorf("template token collision")
		}
	}
	for i, original := range originals {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		token := fmt.Sprintf("%s%d_END", prefix, i)
		result, err := upstream.ReplaceText(editor.Native(), original, token)
		if err != nil {
			return nil, err
		}
		if result.Skipped != 0 || result.Replaced != counts[original] {
			return nil, fmt.Errorf("template key %q has an unsupported text layout", syntax[original])
		}
	}
	for i, original := range originals {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		result, err := upstream.ReplaceText(editor.Native(), fmt.Sprintf("%s%d_END", prefix, i), data[syntax[original]])
		if err != nil {
			return nil, err
		}
		if result.Skipped != 0 || result.Replaced != counts[original] {
			return nil, fmt.Errorf("template value could not be written")
		}
	}
	for key, list := range imageParagraphs {
		image := options.Images[key]
		for _, p := range list {
			if err := editing.AddImage(p, image.Data, image.Width, image.Height); err != nil {
				return nil, err
			}
		}
	}
	var output bytes.Buffer
	if err := editor.Write(ctx, &output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

var keyPattern = regexp.MustCompile("^[A-Za-z_][A-Za-z0-9_]*(?:[.][A-Za-z_][A-Za-z0-9_]*)*$")
var placeholderPattern = regexp.MustCompile("[{][{][[:space:]]*([A-Za-z_][A-Za-z0-9_]*(?:[.][A-Za-z_][A-Za-z0-9_]*)*)[[:space:]]*[}][}]")

func placeholders(text string) (map[string]string, error) {
	if strings.Contains(text, "{%") || strings.Contains(text, "{#") {
		return nil, fmt.Errorf("executable template controls are unsupported")
	}
	out := map[string]string{}
	rest := text
	expressions := 0
	for {
		start := strings.Index(rest, "{{")
		if start < 0 {
			break
		}
		end := strings.Index(rest[start+2:], "}}")
		if end < 0 {
			return nil, fmt.Errorf("unclosed template placeholder")
		}
		end += start + 4
		expressions++
		if expressions > 4096 {
			return nil, fmt.Errorf("template exceeds 4096 placeholders")
		}
		original := rest[start:end]
		match := placeholderPattern.FindStringSubmatch(original)
		if len(match) != 2 || match[0] != original {
			return nil, fmt.Errorf("unsupported template expression")
		}
		out[original] = match[1]
		rest = rest[end:]
	}
	return out, nil
}
func paragraphText(p *etree.Element) string {
	var b strings.Builder
	var walk func(*etree.Element)
	walk = func(e *etree.Element) {
		if e != p && e.Tag == "p" && e.NamespaceURI() == opc.Word {
			return
		}
		if e.Tag == "t" && e.NamespaceURI() == opc.Word {
			b.WriteString(e.Text())
			return
		}
		for _, c := range e.ChildElements() {
			walk(c)
		}
	}
	walk(p)
	return b.String()
}
