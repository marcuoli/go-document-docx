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
	type sequence struct {
		kind string
		list *List
	}
	ids := map[sequence]int{}
	abstract := map[string]int{}
	addInstance := func(abstractID, restartLevel int) int {
		maxNum++
		num := etree.NewElement("w:num")
		num.CreateAttr("w:numId", strconv.Itoa(maxNum))
		num.CreateElement("w:abstractNumId").CreateAttr("w:val", strconv.Itoa(abstractID))
		// Distinct numIds alone may continue the same abstract list in Word.
		// An explicit start override preserves the owned sequence boundary.
		if restartLevel >= 0 {
			override := num.CreateElement("w:lvlOverride")
			override.CreateAttr("w:ilvl", strconv.Itoa(restartLevel))
			override.CreateElement("w:startOverride").CreateAttr("w:val", "1")
		}
		if tail := opc.Child(doc.Root(), opc.Word, "numIdMacAtCleanup"); tail != nil {
			doc.Root().InsertChildAt(tail.Index(), num)
		} else {
			doc.Root().AddChild(num)
		}
		return maxNum
	}
	for _, kind := range []string{"bullet", "decimal"} {
		maxAbstract++
		abstract[kind] = maxAbstract
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
			if child.NamespaceURI() == opc.Word && (child.Tag == "num" || child.Tag == "numIdMacAtCleanup") {
				doc.Root().InsertChildAt(child.Index(), abs)
				inserted = true
				break
			}
		}
		if !inserted {
			doc.Root().AddChild(abs)
		}
		ids[sequence{kind: kind}] = addInstance(maxAbstract, -1)
	}
	// Independent lists share format definitions, not counters. Returning to a
	// parent sequence after nested items must continue that parent, not restart it.
	for _, paragraph := range d.paragraphs {
		if paragraph.sequence == nil {
			continue
		}
		key := sequence{paragraph.listKind, paragraph.sequence}
		if _, exists := ids[key]; exists {
			continue
		}
		ids[key] = addInstance(abstract[key.kind], paragraph.listLevel)
	}
	for _, paragraph := range d.paragraphs {
		if id, ok := ids[sequence{paragraph.listKind, paragraph.sequence}]; ok {
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
