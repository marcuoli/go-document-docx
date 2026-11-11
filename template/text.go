package template

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/internal/opc"
)

// RenderText substitutes strict inline {{key}} values in the original XML parts,
// including split runs, without reconstructing drawings, fields or package assets.
// It preserves unchanged parts byte-for-byte and retains the first character's
// formatting for each replacement. Values are literal and evaluated once.
// Images, controls, field/revision locations and placeholders crossing a break,
// drawing or field boundary are unsupported. Values cannot contain line breaks
// or tabs; use Render or editing for new Word break/tab objects. Neither input
// nor values are mutated; errors and cancellation return no partial document.
func RenderText(ctx context.Context, input []byte, values map[string]string) ([]byte, error) {
	return RenderTextWithOptions(ctx, input, values, TextOptions{})
}

// TextOptions allows explicitly identified simple-field result caches to contain
// placeholders. SimpleFieldResults lists exact instructions after trimming outer
// whitespace. Instructions remain unchanged and are never executed here. A reader
// can overwrite these caches when refreshing fields; callers own that agreement.
// Complex fields and revision text remain unsupported.
type TextOptions struct{ SimpleFieldResults []string }

// RenderTextWithOptions applies RenderText's preservation and bounds while
// permitting caller-identified simple-field result caches. Zero options retain
// RenderText's strict rejection of all field-result placeholders.
func RenderTextWithOptions(ctx context.Context, input []byte, values map[string]string, options TextOptions) ([]byte, error) {
	allowed := map[string]bool{}
	if len(options.SimpleFieldResults) > 128 {
		return nil, fmt.Errorf("too many field cache instructions")
	}
	for _, instruction := range options.SimpleFieldResults {
		if err := opc.Text(instruction); err != nil {
			return nil, err
		}
		instruction = strings.TrimSpace(instruction)
		if instruction == "" || len(instruction) > 4096 {
			return nil, fmt.Errorf("invalid field cache instruction")
		}
		allowed[instruction] = true
	}

	pkg, err := opc.Read(ctx, input)
	if err != nil {
		return nil, err
	}
	totalValues := 0
	for key, value := range values {
		if !keyPattern.MatchString(key) {
			return nil, fmt.Errorf("invalid template key")
		}
		if err := opc.Text(value); err != nil {
			return nil, err
		}
		if strings.ContainsAny(value, "\r\n\t") {
			return nil, fmt.Errorf("template values must be inline text")
		}
		if len(value) > 64<<20-totalValues {
			return nil, fmt.Errorf("template values exceed 64 MiB")
		}
		totalValues += len(value)
	}
	occurrences, textBytes := 0, 0
	for name := range pkg.Parts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !strings.HasPrefix(name, "word/") || !strings.HasSuffix(name, ".xml") {
			continue
		}
		xml, err := pkg.XML(name)
		if err != nil {
			return nil, err
		}
		var excluded strings.Builder
		for _, node := range all(xml.Root()) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			for _, a := range node.Attr {
				if templateSyntax(a.Value) {
					return nil, fmt.Errorf("template syntax in XML attributes is unsupported")
				}
			}
			if node.NamespaceURI() == opc.Word && (node.Tag == "instrText" || node.Tag == "delText" || node.Tag == "delInstrText") {
				excluded.WriteString(node.Text())
			}
			if node.NamespaceURI() == opc.Word && node.Tag == "t" && !hasParagraph(node) && templateSyntax(node.Text()) {
				return nil, fmt.Errorf("template text outside a paragraph is unsupported")
			}
		}
		if templateSyntax(excluded.String()) {
			return nil, fmt.Errorf("template syntax in field/revision text is unsupported")
		}
		fieldText, err := complexFieldText(xml.Root())
		if err != nil {
			return nil, err
		}
		changed := false
		for _, p := range opc.Elements(xml.Root(), opc.Word, "p") {
			leaves := inlineLeaves(p, allowed)
			for i := range leaves {
				leaves[i].forbidden = leaves[i].forbidden || fieldText[leaves[i].node]
			}
			var original strings.Builder
			for _, leaf := range leaves {
				original.WriteString(leaf.node.Text())
			}
			text := original.String()
			if _, err := placeholders(text); err != nil {
				return nil, err
			}
			matches := placeholderPattern.FindAllStringSubmatchIndex(text, -1)
			occurrences += len(matches)
			if occurrences > 4096 {
				return nil, fmt.Errorf("template exceeds 4096 placeholders")
			}
			length := len(text)
			// Reverse splicing keeps original byte offsets valid, including Unicode and
			// several placeholders sharing a run. Inserted values are never rescanned.
			for i := len(matches) - 1; i >= 0; i-- {
				m := matches[i]
				key := text[m[2]:m[3]]
				value, ok := values[key]
				if !ok {
					return nil, fmt.Errorf("missing template key %q", key)
				}
				first, last := -1, -1
				for j, leaf := range leaves {
					if leaf.end > m[0] && leaf.start < m[1] {
						if leaf.forbidden {
							return nil, fmt.Errorf("template placeholder is in field/revision text")
						}
						if first < 0 {
							first = j
						}
						last = j
					}
				}
				if first < 0 || last < 0 || leaves[first].segment != leaves[last].segment {
					return nil, fmt.Errorf("template placeholder crosses a non-text boundary")
				}
				a, b := leaves[first], leaves[last]
				prefix := a.node.Text()[:m[0]-a.start]
				suffix := b.node.Text()[m[1]-b.start:]
				if first == last {
					setInline(a.node, prefix+value+suffix)
				} else {
					setInline(a.node, prefix+value)
					for j := first + 1; j < last; j++ {
						setInline(leaves[j].node, "")
					}
					setInline(b.node, suffix)
				}
				length += len(value) - (m[1] - m[0])
				if length > 64<<20 {
					return nil, fmt.Errorf("template output text exceeds 64 MiB")
				}
				changed = true
			}
			if length > 64<<20-textBytes {
				return nil, fmt.Errorf("template output text exceeds 64 MiB")
			}
			textBytes += length
		}
		if changed {
			if err := pkg.SetXML(name, xml); err != nil {
				return nil, err
			}
		}
	}
	var out bytes.Buffer
	if err := pkg.Write(ctx, &out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
func templateSyntax(text string) bool {
	return strings.Contains(text, "{{") || strings.Contains(text, "{%") || strings.Contains(text, "{#")
}
func hasParagraph(e *etree.Element) bool {
	for p := e.Parent(); p != nil; p = p.Parent() {
		if p.NamespaceURI() == opc.Word && p.Tag == "p" {
			return true
		}
	}
	return false
}

type textLeaf struct {
	node                *etree.Element
	start, end, segment int
	forbidden           bool
}

func inlineLeaves(paragraph *etree.Element, allowed map[string]bool) []textLeaf {
	var leaves []textLeaf
	offset, segment := 0, 0
	var walk func(*etree.Element, bool)
	walk = func(e *etree.Element, forbidden bool) {
		if e != paragraph && e.NamespaceURI() == opc.Word && e.Tag == "p" {
			return
		}
		inlineObject := e.Parent() != nil && e.Parent().NamespaceURI() == opc.Word && e.Parent().Tag == "r" && !(e.NamespaceURI() == opc.Word && (e.Tag == "t" || e.Tag == "rPr"))
		if inlineObject {
			segment++
		}
		if e.NamespaceURI() == opc.Word {
			switch e.Tag {
			case "fldSimple":
				if !allowed[strings.TrimSpace(opc.Attr(e, opc.Word, "instr"))] {
					forbidden = true
				}
				segment++
			case "ins", "del", "moveFrom", "moveTo":
				forbidden = true
				segment++
			case "br", "cr", "tab", "ptab", "sym", "fldChar", "instrText", "drawing", "pict", "object":
				segment++
			}
			if e.Tag == "t" {
				end := offset + len(e.Text())
				leaves = append(leaves, textLeaf{e, offset, end, segment, forbidden})
				offset = end
				return
			}
		}
		for _, c := range e.ChildElements() {
			walk(c, forbidden)
		}
		if inlineObject {
			segment++
		}
		if e.NamespaceURI() == opc.Word {
			switch e.Tag {
			case "fldSimple", "ins", "del", "moveFrom", "moveTo", "drawing", "pict", "object":
				segment++
			}
		}
	}
	forbidden := false
	for parent := paragraph.Parent(); parent != nil; parent = parent.Parent() {
		if parent.NamespaceURI() == opc.Word {
			switch parent.Tag {
			case "fldSimple":
				if !allowed[strings.TrimSpace(opc.Attr(parent, opc.Word, "instr"))] {
					forbidden = true
				}
			case "ins", "del", "moveFrom", "moveTo":
				forbidden = true
			}
		}
	}
	walk(paragraph, forbidden)
	return leaves
}
func setInline(e *etree.Element, text string) { e.SetText(text); e.CreateAttr("xml:space", "preserve") }

// Complex fields may span paragraphs. Text boxes have independent field ranges,
// so their state must not leak into their containing document story.
func complexFieldText(root *etree.Element) (map[*etree.Element]bool, error) {
	marked := map[*etree.Element]bool{}
	var walk func(*etree.Element, *int) error
	walk = func(e *etree.Element, depth *int) error {
		if e.NamespaceURI() == opc.Word && (e.Tag == "txbxContent" || e.Tag == "docPartBody") {
			local := 0
			for _, c := range e.ChildElements() {
				if err := walk(c, &local); err != nil {
					return err
				}
			}
			if local != 0 {
				return fmt.Errorf("unclosed complex field")
			}
			return nil
		}
		if e.NamespaceURI() == opc.Word {
			if e.Tag == "fldChar" {
				switch opc.Attr(e, opc.Word, "fldCharType") {
				case "begin":
					*depth++
				case "end":
					if *depth == 0 {
						return fmt.Errorf("unmatched complex field end")
					}
					*depth--
				}
			}
			if e.Tag == "t" && *depth > 0 {
				marked[e] = true
			}
		}
		for _, c := range e.ChildElements() {
			if err := walk(c, depth); err != nil {
				return err
			}
		}
		return nil
	}
	depth := 0
	if err := walk(root, &depth); err != nil {
		return nil, err
	}
	if depth != 0 {
		return nil, fmt.Errorf("unclosed complex field")
	}
	return marked, nil
}
