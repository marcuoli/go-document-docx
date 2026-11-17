package formatting

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/marcuoli/go-document-docx/internal/opc"
)

// SetParagraphSpaceBefore sets direct space before selected main-document
// paragraphs, including matching paragraphs in tables or text boxes. Twips are
// 1/1440 inch; 0..31680 is accepted. Select 1..256 exact style IDs; an empty ID
// selects paragraphs without an explicit style, as with AlignParagraphs.
//
// Automatic-before spacing is explicitly disabled. Line-based beforeLines in a
// selected paragraph or anywhere in package style/numbering definitions is
// rejected conservatively, without resolving conditional/inherited styles.
// Other spacing attributes, text, fields, fonts, pagination and nonselected
// paragraphs/parts are preserved. Readers control adjacent spacing and page-top
// suppression: this sets a paragraph property, not a fixed number of blank lines.
//
// Input is owned by the caller and never mutated. No match or an already applied
// override returns an exact independent copy. The existing OPC size limits apply;
// errors or cancellation return nil without a partially formatted document.
func SetParagraphSpaceBefore(ctx context.Context, input []byte, twips int, styleIDs ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if twips < 0 || twips > 31680 {
		return nil, fmt.Errorf("paragraph space before must be 0..31680 twips")
	}
	if len(styleIDs) == 0 || len(styleIDs) > 256 {
		return nil, fmt.Errorf("paragraph spacing requires 1-256 style IDs")
	}
	selected := make(map[string]bool, len(styleIDs))
	for _, id := range styleIDs {
		if len(id) > 1024 {
			return nil, fmt.Errorf("paragraph style ID exceeds 1024 bytes")
		}
		if err := opc.Text(id); err != nil {
			return nil, err
		}
		selected[id] = true
	}
	pkg, err := opc.Read(ctx, input)
	if err != nil {
		return nil, err
	}
	doc, err := pkg.XML("word/document.xml")
	if err != nil {
		return nil, err
	}
	opc.WordPrefix(doc.Root())
	body := opc.Child(doc.Root(), opc.Word, "body")
	if body == nil {
		return nil, fmt.Errorf("document body is required")
	}
	changed, checkedDefinitions := false, false
	value := strconv.Itoa(twips)
	for _, paragraph := range opc.Elements(body, opc.Word, "p") {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		id := ""
		pr := opc.Child(paragraph, opc.Word, "pPr")
		if pr != nil {
			if style := opc.Child(pr, opc.Word, "pStyle"); style != nil {
				id = opc.Attr(style, opc.Word, "val")
			}
		}
		if !selected[id] {
			continue
		}
		if !checkedDefinitions {
			// A direct twip value must not silently lose to inherited line units.
			// Reject these definitions rather than alter shared styles/numbering.
			for _, part := range []string{"word/styles.xml", "word/numbering.xml"} {
				if _, ok := pkg.Parts[part]; !ok {
					continue
				}
				definitions, err := pkg.XML(part)
				if err != nil {
					return nil, err
				}
				for _, spacing := range opc.Elements(definitions.Root(), opc.Word, "spacing") {
					if err := ctx.Err(); err != nil {
						return nil, err
					}
					for _, attr := range spacing.Attr {
						if attr.NamespaceURI() == opc.Word && attr.Key == "beforeLines" {
							return nil, fmt.Errorf("line-based before spacing in %s is unsupported", part)
						}
					}
				}
			}
			checkedDefinitions = true
		}
		pr = opc.Properties(paragraph, "pPr")
		current := opc.Child(pr, opc.Word, "spacing")
		if current != nil {
			for _, attr := range current.Attr {
				if attr.NamespaceURI() == opc.Word && attr.Key == "beforeLines" {
					return nil, fmt.Errorf("line-based before spacing on selected paragraph is unsupported")
				}
			}
			auto := opc.Attr(current, opc.Word, "beforeAutospacing")
			if opc.Attr(current, opc.Word, "before") == value && (auto == "0" || auto == "false" || auto == "off") {
				continue
			}
		}
		spacing := opc.Ordered(pr, "spacing", opc.ParagraphOrder)
		opc.SetAttr(spacing, opc.Word, "before", value)
		opc.SetAttr(spacing, opc.Word, "beforeAutospacing", "0")
		changed = true
	}
	if !changed {
		return bytes.Clone(input), nil
	}
	if err := pkg.SetXML("word/document.xml", doc); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := pkg.Write(ctx, &out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
