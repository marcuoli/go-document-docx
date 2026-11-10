package editing

import (
	"fmt"
	"strings"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/internal/opc"
	"github.com/mmonterroca/docxgo/v2/domain"
)

// applyExplicitStyle keeps explicit zero/false overrides that a general writer
// omits as defaults. Read current native values so later supported native edits win.
func applyExplicitStyle(node *etree.Element, run domain.Run, requested TextStyle) {
	pr := properties(node, "rPr")
	if requested.Color != "" {
		c := run.Color()
		opc.Ensure(pr, "color").CreateAttr("w:val", fmt.Sprintf("%02X%02X%02X", c.R, c.G, c.B))
	}
	if requested.Font != "" {
		font := run.Font()
		e := opc.Ensure(pr, "rFonts")
		e.CreateAttr("w:ascii", font.Name)
		e.CreateAttr("w:hAnsi", font.Name)
	}
	if requested.Size != 0 {
		opc.Ensure(pr, "sz").CreateAttr("w:val", fmt.Sprint(run.Size()))
		opc.Ensure(pr, "szCs").CreateAttr("w:val", fmt.Sprint(run.Size()))
	}
	for _, effect := range []struct {
		name      string
		requested *bool
		value     bool
	}{{"b", requested.Bold, run.Bold()}, {"i", requested.Italic, run.Italic()}, {"strike", requested.Strike, run.Strike()}} {
		if effect.requested != nil {
			value := "0"
			if effect.value {
				value = "1"
			}
			opc.Ensure(pr, effect.name).CreateAttr("w:val", value)
		}
	}
	if requested.Underline != nil {
		value := "none"
		if run.Underline() != domain.UnderlineNone {
			value = "single"
		}
		opc.Ensure(pr, "u").CreateAttr("w:val", value)
	}
}
func (p *Paragraph) explicitStyles(node *etree.Element) error {
	raw := []*etree.Element{}
	for _, e := range node.ChildElements() {
		if e.Tag == "r" && e.NamespaceURI() == opc.Word {
			raw = append(raw, e)
		}
	}
	offset := 0
	for i, run := range p.native.Runs() {
		if len(run.Fields()) > 0 {
			for index := range p.styles {
				if index > i {
					return fmt.Errorf("styled text after native fields requires the upstream API")
				}
			}
			if len(p.links) > 0 {
				return fmt.Errorf("field/link layout requires the native hyperlink API")
			}
			continue
		}
		count := strings.Count(run.Text(), "\n") + 1
		if offset+count > len(raw) {
			return fmt.Errorf("unsupported styled run layout")
		}
		if requested, ok := p.styles[i]; ok {
			for _, r := range raw[offset : offset+count] {
				applyExplicitStyle(r, run, requested)
			}
		}
		offset += count
	}
	pr := properties(node, "pPr")
	if p.alignment {
		values := []string{"left", "center", "right", "both", "distribute"}
		a := int(p.native.Alignment())
		if a < 0 || a >= len(values) {
			return fmt.Errorf("invalid paragraph alignment")
		}
		ordered(pr, "jc", paragraphOrder).CreateAttr("w:val", values[a])
	}
	if p.before {
		ordered(pr, "spacing", paragraphOrder).CreateAttr("w:before", fmt.Sprint(p.native.SpacingBefore()))
	}
	if p.after {
		ordered(pr, "spacing", paragraphOrder).CreateAttr("w:after", fmt.Sprint(p.native.SpacingAfter()))
	}
	return nil
}
func mergeStyle(base, over TextStyle) TextStyle {
	if over.Font != "" {
		base.Font = over.Font
	}
	if over.Size != 0 {
		base.Size = over.Size
	}
	if over.Color != "" {
		base.Color = over.Color
	}
	if over.Bold != nil {
		base.Bold = over.Bold
	}
	if over.Italic != nil {
		base.Italic = over.Italic
	}
	if over.Underline != nil {
		base.Underline = over.Underline
	}
	if over.Strike != nil {
		base.Strike = over.Strike
	}
	return base
}
