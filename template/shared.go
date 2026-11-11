package template

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/internal/opc"
)

func (c *composer) shared() error {
	sourceName, err := relatedPart(c.source, "styles", "word/styles.xml")
	if err != nil {
		return err
	}
	destName, err := relatedPart(c.dest, "styles", "word/styles.xml")
	if err != nil {
		return err
	}
	source, err := c.source.XML(sourceName)
	if err != nil {
		return fmt.Errorf("closing styles: %w", err)
	}
	dest, err := c.dest.XML(destName)
	if err != nil {
		return fmt.Errorf("main styles: %w", err)
	}
	for _, style := range source.Root().ChildElements() {
		if style.Tag != "style" || style.NamespaceURI() != opc.Word {
			continue
		}
		id := opc.Attr(style, opc.Word, "styleId")
		if id == "" || c.styleIDs[id] != "" {
			return fmt.Errorf("invalid closing style identity")
		}
		c.styleIDs[id] = c.prefix + id
		if opc.Attr(style, opc.Word, "type") == "paragraph" && (opc.Attr(style, opc.Word, "default") == "1" || opc.Attr(style, opc.Word, "default") == "true") {
			c.defaultStyle = c.styleIDs[id]
		}
	}
	if err := c.numbering(); err != nil {
		return err
	}
	names := existingStyleNames(dest.Root())
	normalizeStyleRuns(dest.Root())
	defaults := opc.Child(source.Root(), opc.Word, "docDefaults")
	for _, style := range source.Root().ChildElements() {
		if style.Tag != "style" || style.NamespaceURI() != opc.Word {
			continue
		}
		cloned := inherited(style, source.Root())
		id := opc.Attr(cloned, opc.Word, "styleId")
		setAttr(cloned, opc.Word, "styleId", c.styleIDs[id])
		for _, a := range append([]etree.Attr(nil), cloned.Attr...) {
			if a.Key == "default" && a.NamespaceURI() == opc.Word {
				key := a.Key
				if a.Space != "" {
					key = a.Space + ":" + a.Key
				}
				cloned.RemoveAttr(key)
			}
		}
		c.rewrite(cloned)
		qualifyStyleNames(cloned, c.prefix, names)
		if c.styleIDs[id] == c.defaultStyle && defaults != nil {
			mergeDefaults(cloned, defaults)
		}
		normalizeStyleRuns(cloned)
		dest.Root().AddChild(cloned)
	}
	if err := c.dest.SetXML(destName, dest); err != nil {
		return err
	}
	return c.fonts()
}
func mergeDefaults(style, defaults *etree.Element) {
	for _, tag := range []string{"pPr", "rPr"} {
		parent := opc.Child(defaults, opc.Word, tag+"Default")
		if parent == nil {
			continue
		}
		values := opc.Child(parent, opc.Word, tag)
		if values == nil {
			continue
		}
		current := opc.Child(style, opc.Word, tag)
		if current == nil {
			placeholder := opc.Ordered(style, tag, opc.StyleOrder)
			style.InsertChildAt(placeholder.Index(), inherited(values, defaults))
			style.RemoveChild(placeholder)
			continue
		}
		order := opc.ParagraphOrder
		if tag == "rPr" {
			order = opc.RunOrder
		}
		for _, value := range values.ChildElements() {
			if opc.Child(current, opc.Word, value.Tag) == nil {
				placeholder := opc.Ordered(current, value.Tag, order)
				current.InsertChildAt(placeholder.Index(), inherited(value, defaults))
				current.RemoveChild(placeholder)
			}
		}
	}
}
func relatedPart(p *opc.Package, kind, fallback string) (string, error) {
	rel, err := p.Relationships("word/document.xml")
	if err != nil {
		return "", err
	}
	for _, e := range rel.Root().ChildElements() {
		if strings.HasSuffix(e.SelectAttrValue("Type", ""), "/"+kind) {
			return opc.Resolve("word/document.xml", e.SelectAttrValue("Target", ""))
		}
	}
	return fallback, nil
}
func (c *composer) numbering() error {
	sourceName, err := relatedPart(c.source, "numbering", "word/numbering.xml")
	if err != nil {
		return err
	}
	if _, ok := c.source.Parts[sourceName]; !ok {
		return nil
	}
	source, err := c.source.XML(sourceName)
	if err != nil {
		return err
	}
	destName, err := relatedPart(c.dest, "numbering", "word/numbering.xml")
	if err != nil {
		return err
	}
	var dest *etree.Document
	if _, ok := c.dest.Parts[destName]; ok {
		dest, err = c.dest.XML(destName)
	} else {
		dest = opc.NewXML("numbering")
	}
	if err != nil {
		return err
	}
	if len(opc.Elements(source.Root(), opc.Word, "numPicBullet")) > 0 || len(opc.Elements(source.Root(), opc.Word, "lvlPicBulletId")) > 0 {
		return fmt.Errorf("closing-page picture bullets are unsupported")
	}
	maxAbstract, maxNum := -1, 0
	for _, e := range opc.Elements(dest.Root(), opc.Word, "abstractNum") {
		n, err := strconv.Atoi(opc.Attr(e, opc.Word, "abstractNumId"))
		if err != nil {
			return err
		}
		maxAbstract = max(maxAbstract, n)
	}
	for _, e := range opc.Elements(dest.Root(), opc.Word, "num") {
		n, err := strconv.Atoi(opc.Attr(e, opc.Word, "numId"))
		if err != nil {
			return err
		}
		maxNum = max(maxNum, n)
	}
	for _, e := range source.Root().ChildElements() {
		switch e.Tag {
		case "abstractNum":
			old := opc.Attr(e, opc.Word, "abstractNumId")
			if old == "" || c.abstractIDs[old] != "" {
				return fmt.Errorf("invalid abstract numbering identity")
			}
			maxAbstract++
			c.abstractIDs[old] = strconv.Itoa(maxAbstract)
		case "num":
			old := opc.Attr(e, opc.Word, "numId")
			if old == "" || old == "0" || c.numIDs[old] != "" {
				return fmt.Errorf("invalid numbering identity")
			}
			maxNum++
			c.numIDs[old] = strconv.Itoa(maxNum)
		}
	}
	for _, e := range source.Root().ChildElements() {
		if e.Tag != "abstractNum" && e.Tag != "num" {
			continue
		}
		cloned := inherited(e, source.Root())
		c.rewrite(cloned)
		if e.Tag == "abstractNum" {
			setAttr(cloned, opc.Word, "abstractNumId", c.abstractIDs[opc.Attr(e, opc.Word, "abstractNumId")])
			inserted := false
			for _, child := range dest.Root().ChildElements() {
				if child.NamespaceURI() == opc.Word && (child.Tag == "num" || child.Tag == "numIdMacAtCleanup") {
					dest.Root().InsertChildAt(child.Index(), cloned)
					inserted = true
					break
				}
			}
			if !inserted {
				dest.Root().AddChild(cloned)
			}
		} else {
			setAttr(cloned, opc.Word, "numId", c.numIDs[opc.Attr(e, opc.Word, "numId")])
			// The cleanup identifier is terminal in CT_Numbering.
			if tail := opc.Child(dest.Root(), opc.Word, "numIdMacAtCleanup"); tail != nil {
				dest.Root().InsertChildAt(tail.Index(), cloned)
			} else {
				dest.Root().AddChild(cloned)
			}
		}
	}
	if err := c.dest.SetXML(destName, dest); err != nil {
		return err
	}
	if err := c.dest.ContentType(destName, "application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"); err != nil {
		return err
	}
	rel, err := c.dest.Relationships("word/document.xml")
	if err != nil {
		return err
	}
	has := false
	for _, e := range rel.Root().ChildElements() {
		if strings.HasSuffix(e.SelectAttrValue("Type", ""), "/numbering") {
			has = true
		}
	}
	if !has {
		opc.AddRelation(rel, "numbering", relative("word/document.xml", destName), "")
	}
	return c.dest.SetXML(opc.RelName("word/document.xml"), rel)
}
func (c *composer) fonts() error {
	sourceName, err := relatedPart(c.source, "fontTable", "word/fontTable.xml")
	if err != nil {
		return err
	}
	if _, ok := c.source.Parts[sourceName]; !ok {
		return nil
	}
	source, err := c.source.XML(sourceName)
	if err != nil {
		return err
	}
	destName, err := relatedPart(c.dest, "fontTable", "word/fontTable.xml")
	if err != nil {
		return err
	}
	var dest *etree.Document
	if _, ok := c.dest.Parts[destName]; ok {
		dest, err = c.dest.XML(destName)
	} else {
		dest = opc.NewXML("fonts")
	}
	if err != nil {
		return err
	}
	rels, err := c.dest.Relationships(destName)
	if err != nil {
		return err
	}
	srcRels, err := c.source.Relationships(sourceName)
	if err != nil {
		return err
	}
	ids, err := c.relationships(sourceName, destName, srcRels, rels, false)
	if err != nil {
		return err
	}
	names := map[string]bool{}
	for _, e := range dest.Root().ChildElements() {
		names[opc.Attr(e, opc.Word, "name")] = true
	}
	for _, font := range source.Root().ChildElements() {
		if font.Tag != "font" || font.NamespaceURI() != opc.Word {
			continue
		}
		name := opc.Attr(font, opc.Word, "name")
		if names[name] {
			continue
		}
		cloned := inherited(font, source.Root())
		for _, e := range all(cloned) {
			for i, a := range e.Attr {
				if a.NamespaceURI() == opc.Office && a.Key == "id" {
					replacement, ok := ids[a.Value]
					if !ok {
						return fmt.Errorf("missing font relationship")
					}
					e.Attr[i].Value = replacement
				}
			}
		}
		dest.Root().AddChild(cloned)
		names[name] = true
	}
	if err := c.dest.SetXML(destName, dest); err != nil {
		return err
	}
	if err := c.dest.SetXML(opc.RelName(destName), rels); err != nil {
		return err
	}
	if err := c.dest.ContentType(destName, "application/vnd.openxmlformats-officedocument.wordprocessingml.fontTable+xml"); err != nil {
		return err
	}
	documentRels, err := c.dest.Relationships("word/document.xml")
	if err != nil {
		return err
	}
	found := false
	for _, relation := range documentRels.Root().ChildElements() {
		if strings.HasSuffix(relation.SelectAttrValue("Type", ""), "/fontTable") {
			found = true
		}
	}
	if !found {
		opc.AddRelation(documentRels, "fontTable", relative("word/document.xml", destName), "")
	}
	return c.dest.SetXML(opc.RelName("word/document.xml"), documentRels)
}
