package editing

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/internal/opc"
	docx "github.com/mmonterroca/docxgo/v2"
)

// Contents appends a native table-of-contents field. Word must update fields to
// compute page numbers; this package does not paginate or evaluate field codes.
func (d *Document) Contents(levels int) (*Paragraph, error) {
	if levels < 1 || levels > 9 {
		return nil, fmt.Errorf("contents levels must be 1 to 9")
	}
	p, err := d.Paragraph("", ParagraphOptions{})
	if err != nil {
		return nil, err
	}
	r, err := p.native.AddRun()
	if err != nil {
		return nil, err
	}
	f := docx.NewTOCField(nil)
	if err := f.SetCode(fmt.Sprintf("TOC \\o \"1-%d\" \\h \\z \\u", levels)); err != nil {
		return nil, err
	}
	if err := r.AddField(f); err != nil {
		return nil, err
	}
	return p, nil
}

func (d *Document) numbering(p *opc.Package, rels *etree.Document, nodes map[*Paragraph]*etree.Element) error {
	needed := false
	for _, paragraph := range d.paragraphs {
		if paragraph.listKind == "bullet" || paragraph.listKind == "decimal" {
			needed = true
		}
	}
	if !needed {
		return nil
	}
	name := "word/numbering.xml"
	hasRelation := false
	for _, rel := range rels.Root().ChildElements() {
		if strings.HasSuffix(rel.SelectAttrValue("Type", ""), "/numbering") {
			var err error
			name, err = opc.Resolve("word/document.xml", rel.SelectAttrValue("Target", ""))
			if err != nil {
				return err
			}
			hasRelation = true
		}
	}
	var doc *etree.Document
	var err error
	if _, ok := p.Parts[name]; ok {
		doc, err = p.XML(name)
	} else {
		doc = opc.NewXML("numbering")
	}
	if err != nil {
		return err
	}
	opc.WordPrefix(doc.Root())
	maxAbstract, maxNum := -1, 0
	for _, e := range opc.Elements(doc.Root(), opc.Word, "abstractNum") {
		i, err := strconv.Atoi(opc.Attr(e, opc.Word, "abstractNumId"))
		if err != nil {
			return err
		}
		maxAbstract = max(maxAbstract, i)
	}
	for _, e := range opc.Elements(doc.Root(), opc.Word, "num") {
		i, err := strconv.Atoi(opc.Attr(e, opc.Word, "numId"))
		if err != nil {
			return err
		}
		maxNum = max(maxNum, i)
	}
	ids := map[string]int{}
	for _, kind := range []string{"bullet", "decimal"} {
		maxAbstract++
		maxNum++
		ids[kind] = maxNum
		abs := etree.NewElement("w:abstractNum")
		abs.CreateAttr("w:abstractNumId", strconv.Itoa(maxAbstract))
		abs.CreateElement("w:multiLevelType").CreateAttr("w:val", "multilevel")
		for level := 0; level < 9; level++ {
			lvl := abs.CreateElement("w:lvl")
			lvl.CreateAttr("w:ilvl", strconv.Itoa(level))
			lvl.CreateElement("w:start").CreateAttr("w:val", "1")
			lvl.CreateElement("w:numFmt").CreateAttr("w:val", kind)
			value := "•"
			if kind == "decimal" {
				value = fmt.Sprintf("%%%d.", level+1)
			}
			lvl.CreateElement("w:lvlText").CreateAttr("w:val", value)
			lvl.CreateElement("w:lvlJc").CreateAttr("w:val", "left")
			ind := lvl.CreateElement("w:pPr").CreateElement("w:ind")
			ind.CreateAttr("w:left", strconv.Itoa((level+1)*720))
			ind.CreateAttr("w:hanging", "360")
		}
		inserted := false
		for _, child := range doc.Root().ChildElements() {
			if child.Tag == "num" {
				doc.Root().InsertChildAt(child.Index(), abs)
				inserted = true
				break
			}
		}
		if !inserted {
			doc.Root().AddChild(abs)
		}
		num := doc.Root().CreateElement("w:num")
		num.CreateAttr("w:numId", strconv.Itoa(maxNum))
		num.CreateElement("w:abstractNumId").CreateAttr("w:val", strconv.Itoa(maxAbstract))
	}
	for _, paragraph := range d.paragraphs {
		if id, ok := ids[paragraph.listKind]; ok {
			pr := properties(nodes[paragraph], "pPr")
			num := ordered(pr, "numPr", paragraphOrder)
			num.CreateElement("w:ilvl").CreateAttr("w:val", strconv.Itoa(paragraph.listLevel))
			num.CreateElement("w:numId").CreateAttr("w:val", strconv.Itoa(id))
		}
	}
	if err := p.SetXML(name, doc); err != nil {
		return err
	}
	if err := p.ContentType(name, "application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"); err != nil {
		return err
	}
	if !hasRelation {
		opc.AddRelation(rels, "numbering", "numbering.xml", "")
	}
	return nil
}
