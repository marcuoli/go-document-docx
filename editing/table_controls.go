package editing

import (
	"fmt"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/internal/opc"
	"github.com/mmonterroca/docxgo/v2/domain"
)

type cellAlignment struct{ horizontal, vertical bool }
type rowHeight struct {
	twips int
	exact bool
}

// Align sets explicit horizontal/vertical cell alignment. Nil inherits the current
// setting; non-nil zero means left/top and overrides inherited template formatting.
func (t *Table) Align(row, column int, horizontal *domain.Alignment, vertical *domain.VerticalAlignment) error {
	r, err := t.native.Row(row)
	if err != nil {
		return err
	}
	cell, err := r.Cell(column)
	if err != nil {
		return err
	}
	if horizontal != nil {
		for _, p := range cell.Paragraphs() {
			if err := p.SetAlignment(*horizontal); err != nil {
				return err
			}
		}
	}
	if vertical != nil {
		if err := cell.SetVerticalAlignment(*vertical); err != nil {
			return err
		}
	}
	if t.alignments == nil {
		t.alignments = map[[2]int]cellAlignment{}
	}
	key := [2]int{row, column}
	existing := t.alignments[key]
	existing.horizontal = existing.horizontal || horizontal != nil
	existing.vertical = existing.vertical || vertical != nil
	t.alignments[key] = existing
	return nil
}

// Height sets a row's height in twips. Zero with exact=false means automatic.
// Exact heights can clip tall content; use at-least heights for arbitrary evidence.
func (t *Table) Height(row, twips int, exact bool) error {
	if twips < 0 || twips > 31680 || exact && twips == 0 {
		return fmt.Errorf("invalid row height")
	}
	r, err := t.native.Row(row)
	if err != nil {
		return err
	}
	if err := r.SetHeight(twips); err != nil {
		return err
	}
	if t.heights == nil {
		t.heights = map[int]rowHeight{}
	}
	t.heights[row] = rowHeight{twips, exact}
	return nil
}
func (t *Table) rowControls(row *etree.Element, index int) error {
	native, err := t.native.Row(index)
	if err != nil {
		return err
	}
	pr := properties(row, "trPr")
	for _, flag := range []struct {
		tag   string
		value bool
	}{{"cantSplit", native.CantSplit()}, {"tblHeader", native.RepeatAsHeader()}} {
		value := "0"
		if flag.value {
			value = "1"
		}
		opc.Ensure(pr, flag.tag).CreateAttr("w:val", value)
	}
	if height, ok := t.heights[index]; ok {
		e := opc.Ensure(pr, "trHeight")
		e.CreateAttr("w:val", fmt.Sprint(height.twips))
		rule := "atLeast"
		if height.exact {
			rule = "exact"
		} else if height.twips == 0 {
			rule = "auto"
		}
		e.CreateAttr("w:hRule", rule)
	}
	return nil
}
func (t *Table) cellControls(cell *etree.Element, row, column int) error {
	requested := t.alignments[[2]int{row, column}]
	requested.horizontal = requested.horizontal || t.options.Columns[column].Alignment != nil
	if !requested.horizontal && !requested.vertical {
		return nil
	}
	nativeRow, err := t.native.Row(row)
	if err != nil {
		return err
	}
	native, err := nativeRow.Cell(column)
	if err != nil {
		return err
	}
	if requested.horizontal {
		paragraphs := native.Paragraphs()
		var raw []*etree.Element
		for _, child := range cell.ChildElements() {
			if child.NamespaceURI() == opc.Word && child.Tag == "p" {
				raw = append(raw, child)
			}
		}
		if len(paragraphs) == 0 || len(raw) != len(paragraphs) {
			return fmt.Errorf("unsupported aligned cell paragraph layout")
		}
		// Every represented paragraph needs the explicit zero-valued override;
		// otherwise later paragraphs can still inherit centered template styles.
		values := []string{"left", "center", "right", "both", "distribute"}
		for i, paragraph := range paragraphs {
			value := int(paragraph.Alignment())
			if value < 0 || value >= len(values) {
				return fmt.Errorf("invalid cell alignment")
			}
			ordered(properties(raw[i], "pPr"), "jc", paragraphOrder).CreateAttr("w:val", values[value])
		}
	}
	if requested.vertical {
		values := []string{"top", "center", "bottom"}
		value := int(native.VerticalAlignment())
		if value < 0 || value >= len(values) {
			return fmt.Errorf("invalid vertical alignment")
		}
		ordered(properties(cell, "tcPr"), "vAlign", cellOrder).CreateAttr("w:val", values[value])
	}
	return nil
}
