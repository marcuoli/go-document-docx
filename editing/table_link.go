package editing

import (
	"fmt"
	"strings"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/internal/opc"
)

// Link replaces one unmerged cell's literal value with a styled hyperlink.
// Internal targets name existing bookmarks; external links use explicit web/mail
// schemes and are never fetched. A previous shortening tooltip is removed.
func (t *Table) Link(row, column int, text, target, tooltip string, internal bool) error {
	for _, value := range []string{text, target, tooltip} {
		if err := validText(value); err != nil {
			return err
		}
	}
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("link target is required")
	}
	if !internal && !strings.HasPrefix(target, "https://") && !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "mailto:") {
		return fmt.Errorf("unsupported external link scheme")
	}
	r, err := t.native.Row(row)
	if err != nil {
		return err
	}
	cell, err := r.Cell(column)
	if err != nil {
		return err
	}
	paragraphs := cell.Paragraphs()
	if len(paragraphs) != 1 || len(paragraphs[0].Runs()) != 1 || cell.GridSpan() > 1 {
		return fmt.Errorf("cell link requires a simple unmerged cell")
	}
	run := paragraphs[0].Runs()[0]
	if err := run.SetText(text); err != nil {
		return err
	}
	style := TextStyle{Color: "315357", Underline: new(true)}
	if err := styleRun(run, style); err != nil {
		return err
	}
	if t.links == nil {
		t.links = map[[2]int]link{}
	}
	key := [2]int{row, column}
	t.links[key] = link{target: target, text: text, tooltip: tooltip, internal: internal}
	delete(t.tips, key)
	delete(t.displays, key)
	t.styles[key] = mergeStyle(t.styles[key], style)
	return nil
}
func cellLink(cell *etree.Element, link link, relations *etree.Document, names map[string]bool) error {
	paragraphs := opc.Elements(cell, opc.Word, "p")
	if len(paragraphs) != 1 {
		return fmt.Errorf("unsupported linked-cell layout")
	}
	p := paragraphs[0]
	runs := []*etree.Element{}
	for _, e := range p.ChildElements() {
		if e.Tag == "r" && e.NamespaceURI() == opc.Word {
			runs = append(runs, e)
		}
	}
	if len(runs) == 0 {
		return fmt.Errorf("linked cell has no runs")
	}
	hyperlink := etree.NewElement("w:hyperlink")
	if link.internal {
		if !names[link.target] {
			return fmt.Errorf("missing cell-link bookmark %q", link.target)
		}
		hyperlink.CreateAttr("w:anchor", link.target)
	} else {
		hyperlink.CreateAttr("r:id", opc.AddRelation(relations, "hyperlink", link.target, "External"))
	}
	if link.tooltip != "" {
		hyperlink.CreateAttr("w:tooltip", link.tooltip)
	}
	position := runs[0].Index()
	for _, r := range runs {
		p.RemoveChild(r)
		hyperlink.AddChild(r)
	}
	p.InsertChildAt(position, hyperlink)
	return nil
}
