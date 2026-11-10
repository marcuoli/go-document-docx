package editing

import (
	"bytes"
	"context"
	"fmt"

	"github.com/marcuoli/go-document-docx/internal/opc"
	docx "github.com/mmonterroca/docxgo/v2"
)

// A loaded template's style part is retained verbatim by the general writer.
// Serialize only helper-owned new styles through a fresh upstream document and
// append those definitions to the retained part. Existing template styles remain
// untouched; no third-party code or module cache is modified.
func (d *Document) amendStyles(ctx context.Context, p *opc.Package) error {
	if len(d.styles) == 0 {
		return nil
	}
	fresh := docx.NewDocument()
	for _, style := range d.styles {
		current, err := d.native.StyleManager().GetStyle(style.ID())
		if err != nil || current != style {
			return fmt.Errorf("owned style no longer belongs to document")
		}
		if err := fresh.StyleManager().AddStyle(style); err != nil {
			return err
		}
	}
	// The temporary document is only an upstream serializer input, never published.
	paragraph, err := fresh.AddParagraph()
	if err != nil {
		return err
	}
	run, err := paragraph.AddRun()
	if err != nil {
		return err
	}
	if err := run.SetText("Style definitions"); err != nil {
		return err
	}
	var b bytes.Buffer
	if _, err := fresh.WriteTo(opc.Writer{Ctx: ctx, Destination: &b}); err != nil {
		return err
	}
	exported, err := opc.Read(ctx, b.Bytes())
	if err != nil {
		return err
	}
	definitions, err := exported.XML("word/styles.xml")
	if err != nil {
		return err
	}
	target, err := p.XML("word/styles.xml")
	if err != nil {
		return err
	}
	opc.WordPrefix(target.Root())
	for _, style := range d.styles {
		for _, existing := range opc.Elements(target.Root(), opc.Word, "style") {
			if opc.Attr(existing, opc.Word, "styleId") == style.ID() {
				return fmt.Errorf("new style conflicts with retained template style")
			}
		}
		found := false
		for _, candidate := range opc.Elements(definitions.Root(), opc.Word, "style") {
			if opc.Attr(candidate, opc.Word, "styleId") == style.ID() {
				cloned := candidate.Copy()
				opc.WordPrefix(cloned)
				rpr := opc.Ordered(cloned, "rPr", opc.StyleOrder)
				family := style.Font().Name
				opc.Ordered(rpr, "rFonts", opc.RunOrder).CreateAttr("w:ascii", family)
				opc.Ordered(rpr, "rFonts", opc.RunOrder).CreateAttr("w:hAnsi", family)
				opc.Ordered(rpr, "sz", opc.RunOrder).CreateAttr("w:val", fmt.Sprint(style.Size()))
				opc.Ordered(rpr, "szCs", opc.RunOrder).CreateAttr("w:val", fmt.Sprint(style.Size()))
				target.Root().AddChild(cloned)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("upstream did not serialize the new style")
		}
	}
	return p.SetXML("word/styles.xml", target)
}
