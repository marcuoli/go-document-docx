package editing

import (
	"fmt"
	"github.com/mmonterroca/docxgo/v2/domain"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/internal/opc"
)

func (t *Table) amend(node *etree.Element, id *int, names map[string]bool, relations *etree.Document) error {
	normalizeText(node)
	pr := properties(node, "tblPr")
	layout := ordered(pr, "tblLayout", tableOrder)
	layout.CreateAttr("w:type", "fixed")
	width := 0
	for _, w := range t.widths {
		width += w
	}
	preferred := ordered(pr, "tblW", tableOrder)
	preferred.CreateAttr("w:w", strconv.Itoa(width))
	preferred.CreateAttr("w:type", "dxa")
	indent := ordered(pr, "tblInd", tableOrder)
	indent.CreateAttr("w:w", strconv.Itoa(t.options.Indent))
	indent.CreateAttr("w:type", "dxa")
	margins := Margins{Top: 70, Left: 108, Bottom: 70, Right: 108}
	if t.options.Margins != nil {
		margins = *t.options.Margins
	}
	setMargins(ordered(pr, "tblCellMar", tableOrder), margins)
	grid := opc.Child(node, opc.Word, "tblGrid")
	if grid == nil {
		grid = etree.NewElement("w:tblGrid")
		node.InsertChildAt(pr.Index()+1, grid)
	}
	for _, child := range grid.ChildElements() {
		grid.RemoveChild(child)
	}
	for _, w := range t.widths {
		grid.CreateElement("w:gridCol").CreateAttr("w:w", strconv.Itoa(w))
	}
	rowIndex := 0
	for _, row := range node.ChildElements() {
		if row.Tag != "tr" || row.NamespaceURI() != opc.Word {
			continue
		}
		if err := t.rowControls(row, rowIndex); err != nil {
			return err
		}
		col := 0
		for _, cell := range row.ChildElements() {
			if cell.Tag != "tc" || cell.NamespaceURI() != opc.Word {
				continue
			}
			cp := properties(cell, "tcPr")
			span := 1
			if s := opc.Child(cp, opc.Word, "gridSpan"); s != nil {
				var err error
				span, err = strconv.Atoi(opc.Attr(s, opc.Word, "val"))
				if err != nil || span < 1 {
					return fmt.Errorf("invalid cell span")
				}
			}
			if col+span > len(t.widths) {
				return fmt.Errorf("cell span exceeds table grid")
			}
			w := 0
			for _, v := range t.widths[col : col+span] {
				w += v
			}
			cw := ordered(cp, "tcW", cellOrder)
			cw.CreateAttr("w:w", strconv.Itoa(w))
			cw.CreateAttr("w:type", "dxa")
			noWrap := false
			for _, c := range t.options.Columns[col : col+span] {
				noWrap = noWrap || c.SingleLine
			}
			if noWrap {
				ordered(cp, "noWrap", cellOrder).CreateAttr("w:val", "1")
				ordered(cp, "tcFitText", cellOrder).CreateAttr("w:val", "0")
			}
			for offset := 0; offset < span; offset++ {
				key := [2]int{rowIndex, col + offset}
				if offset == 0 {
					if err := t.cellControls(cell, rowIndex, col); err != nil {
						return err
					}
					nativeRow, err := t.native.Row(rowIndex)
					if err != nil {
						return err
					}
					nativeCell, err := nativeRow.Cell(col)
					if err != nil {
						return err
					}
					nativeRuns := []domain.Run{}
					for _, p := range nativeCell.Paragraphs() {
						nativeRuns = append(nativeRuns, p.Runs()...)
					}
					rawRuns := opc.Elements(cell, opc.Word, "r")
					position := 0
					for _, run := range nativeRuns {
						if len(run.Fields()) > 0 {
							return fmt.Errorf("field-bearing table cells require the upstream API")
						}
						count := strings.Count(run.Text(), "\n") + 1
						if position+count > len(rawRuns) {
							return fmt.Errorf("unsupported table run layout")
						}
						for _, raw := range rawRuns[position : position+count] {
							applyExplicitStyle(raw, run, t.styles[key])
						}
						position += count
					}
				}
				if m, ok := t.cellMargins[key]; ok {
					setMargins(ordered(cp, "tcMar", cellOrder), m)
				}
				if full, ok := t.tips[key]; ok {
					if span != 1 {
						return fmt.Errorf("display shortening is unsupported on merged cells")
					}
					paragraphs := opc.Elements(cell, opc.Word, "p")
					if len(paragraphs) != 1 {
						return fmt.Errorf("shortened cell has unsupported paragraph layout")
					}
					p := paragraphs[0]
					runs := []*etree.Element{}
					for _, e := range p.ChildElements() {
						if e.Tag == "r" && e.NamespaceURI() == opc.Word {
							runs = append(runs, e)
						}
					}
					if len(runs) != 1 {
						return fmt.Errorf("shortened cell has unsupported run layout")
					}
					text := ""
					for _, e := range opc.Elements(runs[0], opc.Word, "t") {
						text += e.Text()
					}
					if text != t.displays[key] {
						return fmt.Errorf("shortened cell changed after tooltip capture")
					}
					name := ""
					for {
						name = fmt.Sprintf("cell_value_%d", *id+1)
						*id += 1
						if !names[name] {
							break
						}
					}
					names[name] = true
					start := etree.NewElement("w:bookmarkStart")
					start.CreateAttr("w:id", strconv.Itoa(*id))
					start.CreateAttr("w:name", name)
					link := etree.NewElement("w:hyperlink")
					link.CreateAttr("w:anchor", name)
					link.CreateAttr("w:tooltip", full)
					run := runs[0]
					position := run.Index()
					p.RemoveChild(run)
					link.AddChild(run)
					p.InsertChildAt(position, start)
					p.InsertChildAt(position+1, link)
					end := p.CreateElement("w:bookmarkEnd")
					end.CreateAttr("w:id", strconv.Itoa(*id))
				}
			}
			if intent, ok := t.links[[2]int{rowIndex, col}]; ok {
				if span != 1 {
					return fmt.Errorf("cell links require unmerged cells")
				}
				if err := cellLink(cell, intent, relations, names); err != nil {
					return err
				}
			}
			col += span
		}
		if col != len(t.widths) {
			return fmt.Errorf("table row does not cover its grid")
		}
		rowIndex++
	}
	return nil
}
func setMargins(parent *etree.Element, m Margins) {
	for _, side := range []struct {
		name  string
		value int
	}{{"top", m.Top}, {"left", m.Left}, {"bottom", m.Bottom}, {"right", m.Right}} {
		e := opc.Child(parent, opc.Word, side.name)
		if e == nil {
			e = parent.CreateElement("w:" + side.name)
		}
		e.CreateAttr("w:w", strconv.Itoa(side.value))
		e.CreateAttr("w:type", "dxa")
	}
}
