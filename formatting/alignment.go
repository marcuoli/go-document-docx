// Package formatting applies targeted changes to retained document packages.
package formatting

import (
	"bytes"
	"context"
	"fmt"

	"github.com/marcuoli/go-document-docx/internal/opc"
)

// AlignParagraphs applies direct alignment to paragraphs with selected style IDs
// in the main document, including matching paragraphs in tables or text boxes.
// Alignment is left, center, right or justify. At least one exact style ID is
// required; an empty ID selects paragraphs without an explicit paragraph style.
// Other styles, headers, footers, fields and text remain unchanged. Input is
// copied, unchanged package parts retain their bytes, and a no-op returns an
// exact copy. Errors or cancellation return no partially formatted document.
func AlignParagraphs(ctx context.Context, input []byte, alignment string, styleIDs ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch alignment {
	case "left", "center", "right":
	case "justify":
		alignment = "both"
	default:
		return nil, fmt.Errorf("unsupported paragraph alignment")
	}
	if len(styleIDs) == 0 || len(styleIDs) > 256 {
		return nil, fmt.Errorf("paragraph alignment requires 1-256 style IDs")
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
	xml, err := pkg.XML("word/document.xml")
	if err != nil {
		return nil, err
	}
	opc.WordPrefix(xml.Root())
	body := opc.Child(xml.Root(), opc.Word, "body")
	if body == nil {
		return nil, fmt.Errorf("document body is required")
	}
	changed := false
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
		pr = opc.Properties(paragraph, "pPr")
		current := opc.Child(pr, opc.Word, "jc")
		if current != nil && opc.Attr(current, opc.Word, "val") == alignment {
			continue
		}
		// Insert within the schema's paragraph-property order. Direct alignment
		// avoids changing inherited heading/table styles or field instructions.
		opc.SetAttr(opc.Ordered(pr, "jc", opc.ParagraphOrder), opc.Word, "val", alignment)
		changed = true
	}
	if !changed {
		return bytes.Clone(input), nil
	}
	if err := pkg.SetXML("word/document.xml", xml); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := pkg.Write(ctx, &out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
