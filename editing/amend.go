package editing

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/internal/opc"
)

var paragraphOrder = opc.ParagraphOrder
var tableOrder = opc.TableOrder
var cellOrder = opc.CellOrder

func ordered(parent *etree.Element, tag string, order []string) *etree.Element {
	return opc.Ordered(parent, tag, order)
}
func properties(parent *etree.Element, tag string) *etree.Element { return opc.Properties(parent, tag) }
func (d *Document) amend(ctx context.Context, p *opc.Package) error {
	xml, err := p.XML("word/document.xml")
	if err != nil {
		return err
	}
	opc.WordPrefix(xml.Root())
	body := opc.Child(xml.Root(), opc.Word, "body")
	if body == nil {
		return fmt.Errorf("missing document body")
	}
	paragraphs, tables := []*etree.Element{}, []*etree.Element{}
	for _, e := range body.ChildElements() {
		if e.NamespaceURI() == opc.Word {
			switch e.Tag {
			case "p":
				if pr := opc.Child(e, opc.Word, "pPr"); len(e.ChildElements()) == 1 && pr != nil && len(pr.ChildElements()) == 1 && opc.Child(pr, opc.Word, "sectPr") != nil {
					continue
				}
				paragraphs = append(paragraphs, e)
			case "tbl":
				tables = append(tables, e)
			}
		}
	}
	natives := d.native.Paragraphs()
	nativeTables := d.native.Tables()
	if len(paragraphs) != len(natives) || len(tables) != len(nativeTables) {
		return fmt.Errorf("unsupported body/model layout")
	}
	relations, err := p.Relationships("word/document.xml")
	if err != nil {
		return err
	}
	names := map[string]bool{}
	maxID := 0
	for _, start := range opc.Elements(xml.Root(), opc.Word, "bookmarkStart") {
		name := opc.Attr(start, opc.Word, "name")
		if names[name] {
			return fmt.Errorf("duplicate bookmark name")
		}
		names[name] = true
	}
	for _, tag := range []string{"bookmarkStart", "bookmarkEnd"} {
		for _, e := range opc.Elements(xml.Root(), opc.Word, tag) {
			id, err := strconv.Atoi(opc.Attr(e, opc.Word, "id"))
			if err != nil || id < 0 {
				return fmt.Errorf("invalid bookmark ID")
			}
			maxID = max(maxID, id)
		}
	}
	ownNodes := map[*Paragraph]*etree.Element{}
	for _, paragraph := range d.paragraphs {
		if err := ctx.Err(); err != nil {
			return err
		}
		index := -1
		for i, n := range natives {
			if n == paragraph.native {
				index = i
				break
			}
		}
		if index < 0 {
			return fmt.Errorf("paragraph helper no longer belongs to document")
		}
		node := paragraphs[index]
		ownNodes[paragraph] = node
		normalizeText(node)
		if err := paragraph.explicitStyles(node); err != nil {
			return err
		}
		if paragraph.bookmark != "" {
			if names[paragraph.bookmark] {
				return fmt.Errorf("duplicate bookmark %q", paragraph.bookmark)
			}
			names[paragraph.bookmark] = true
			maxID++
			start := etree.NewElement("w:bookmarkStart")
			start.CreateAttr("w:id", strconv.Itoa(maxID))
			start.CreateAttr("w:name", paragraph.bookmark)
			index := 0
			if len(node.ChildElements()) > 0 && node.ChildElements()[0].Tag == "pPr" {
				index = node.ChildElements()[0].Index() + 1
			}
			node.InsertChildAt(index, start)
			end := node.CreateElement("w:bookmarkEnd")
			end.CreateAttr("w:id", strconv.Itoa(maxID))
		}
		if paragraph.listKind == "heading" {
			pr := properties(node, "pPr")
			ordered(pr, "outlineLvl", paragraphOrder).CreateAttr("w:val", strconv.Itoa(paragraph.listLevel))
		}
	}
	for _, paragraph := range d.paragraphs {
		node := ownNodes[paragraph]
		runs := []*etree.Element{}
		for _, e := range node.ChildElements() {
			if e.Tag == "r" && e.NamespaceURI() == opc.Word {
				runs = append(runs, e)
			}
		}
		// The released serializer expands a newline-bearing native run into several
		// XML runs. Track that expansion instead of binding links by raw XML index.
		offsets, counts := []int{}, []int{}
		expected := 0
		for _, run := range paragraph.native.Runs() {
			if len(paragraph.links) > 0 && len(run.Fields()) > 0 {
				return fmt.Errorf("links with field-bearing runs require the upstream native hyperlink API")
			}
			count := strings.Count(run.Text(), "\n") + 1
			offsets = append(offsets, expected)
			counts = append(counts, count)
			expected += count
		}
		if len(paragraph.links) > 0 && len(runs) != expected {
			return fmt.Errorf("unsupported link/run layout")
		}
		for _, link := range paragraph.links {
			if link.run < 0 || link.run >= len(offsets) {
				return fmt.Errorf("link run no longer exists")
			}
			e := etree.NewElement("w:hyperlink")
			if link.internal {
				if !names[link.target] {
					return fmt.Errorf("missing bookmark target %q", link.target)
				}
				e.CreateAttr("w:anchor", link.target)
			} else {
				e.CreateAttr("r:id", opc.AddRelation(relations, "hyperlink", link.target, "External"))
			}
			if link.tooltip != "" {
				e.CreateAttr("w:tooltip", link.tooltip)
			}
			group := runs[offsets[link.run] : offsets[link.run]+counts[link.run]]
			position := group[0].Index()
			for _, r := range group {
				node.RemoveChild(r)
				e.AddChild(r)
			}
			node.InsertChildAt(position, e)
		}
	}
	for _, table := range d.tables {
		if err := ctx.Err(); err != nil {
			return err
		}
		index := -1
		for i, n := range nativeTables {
			if n == table.native {
				index = i
				break
			}
		}
		if index < 0 {
			return fmt.Errorf("table helper no longer belongs to document")
		}
		if err := table.amend(tables[index], &maxID, names, relations); err != nil {
			return err
		}
	}
	if err := d.numbering(p, relations, ownNodes); err != nil {
		return err
	}
	for index, section := range opc.Elements(body, opc.Word, "sectPr") {
		if index < len(d.gutters) {
			if margins := opc.Child(section, opc.Word, "pgMar"); margins != nil {
				margins.CreateAttr("w:gutter", strconv.Itoa(d.gutters[index]))
			}
		}
	}
	if err := p.SetXML("word/document.xml", xml); err != nil {
		return err
	}
	if err := p.SetXML(opc.RelName("word/document.xml"), relations); err != nil {
		return err
	}
	return compatibility(p)
}

// normalizeText represents explicit newlines/tabs as Word elements without
// interpreting literal values as markup or modifying an upstream source file.
func normalizeText(parent *etree.Element) {
	for _, t := range opc.Elements(parent, opc.Word, "t") {
		text := t.Text()
		if !strings.ContainsAny(text, "\r\n\t") {
			continue
		}
		container := t.Parent()
		position := t.Index()
		container.RemoveChild(t)
		text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
		start := 0
		for i, r := range text {
			if r != '\n' && r != '\t' {
				continue
			}
			node := etree.NewElement("w:t")
			node.CreateAttr("xml:space", "preserve")
			node.SetText(text[start:i])
			container.InsertChildAt(position, node)
			position++
			tag := "br"
			if r == '\t' {
				tag = "tab"
			}
			container.InsertChildAt(position, etree.NewElement("w:"+tag))
			position++
			start = i + 1
		}
		node := etree.NewElement("w:t")
		node.CreateAttr("xml:space", "preserve")
		node.SetText(text[start:])
		container.InsertChildAt(position, node)
	}
}

func compatibility(p *opc.Package) error {
	var settings *etree.Document
	var err error
	if _, ok := p.Parts["word/settings.xml"]; ok {
		settings, err = p.XML("word/settings.xml")
	} else {
		settings = opc.NewXML("settings")
	}
	if err != nil {
		return err
	}
	opc.WordPrefix(settings.Root())
	compat := opc.Child(settings.Root(), opc.Word, "compat")
	if compat == nil {
		compat = etree.NewElement("w:compat")
		inserted := false
		later := map[string]bool{"docVars": true, "rsids": true, "mathPr": true, "themeFontLang": true, "clrSchemeMapping": true, "decimalSymbol": true, "listSeparator": true}
		for _, e := range settings.Root().ChildElements() {
			if later[e.Tag] {
				settings.Root().InsertChildAt(e.Index(), compat)
				inserted = true
				break
			}
		}
		if !inserted {
			settings.Root().AddChild(compat)
		}
	}
	flags := opc.Elements(compat, opc.Word, "doNotExpandShiftReturn")
	if len(flags) == 0 {
		flags = append(flags, compat.CreateElement("w:doNotExpandShiftReturn"))
	}
	flags[0].CreateAttr("w:val", "1")
	for _, duplicate := range flags[1:] {
		duplicate.Parent().RemoveChild(duplicate)
	}
	update := opc.Child(settings.Root(), opc.Word, "updateFields")
	if update == nil {
		update = etree.NewElement("w:updateFields")
		settings.Root().InsertChildAt(0, update)
	}
	update.CreateAttr("w:val", "true")
	if err := p.SetXML("word/settings.xml", settings); err != nil {
		return err
	}
	if err := p.ContentType("word/settings.xml", "application/vnd.openxmlformats-officedocument.wordprocessingml.settings+xml"); err != nil {
		return err
	}
	rel, err := p.Relationships("word/document.xml")
	if err != nil {
		return err
	}
	found := false
	for _, r := range rel.Root().ChildElements() {
		if strings.HasSuffix(r.SelectAttrValue("Type", ""), "/settings") {
			found = true
		}
	}
	if !found {
		opc.AddRelation(rel, "settings", "settings.xml", "")
	}
	return p.SetXML(opc.RelName("word/document.xml"), rel)
}
