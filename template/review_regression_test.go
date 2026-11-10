package template

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/fonts"
	"github.com/marcuoli/go-document-docx/internal/opc"
	"github.com/marcuoli/go-document-docx/report"
	docx "github.com/mmonterroca/docxgo/v2"
)

func blankPackage(t *testing.T, text string) *opc.Package {
	t.Helper()
	d, _ := editing.New(report.Defaults())
	_, _ = d.Paragraph(text, editing.ParagraphOptions{})
	var b bytes.Buffer
	if err := d.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	p, err := opc.Read(context.Background(), b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func packageBytes(t *testing.T, p *opc.Package) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := p.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestClosingBodyLocalNamespaceSurvives(t *testing.T) {
	base := blankPackage(t, "Main")
	closing := blankPackage(t, "Closing")
	closing.Parts["word/document.xml"] = []byte("<w:document xmlns:w=\"" + opc.Word + "\"><w:body xmlns:alias=\"" + opc.Word + "\"><alias:p><alias:r><alias:t>Local namespace</alias:t></alias:r></alias:p><w:sectPr/></w:body></w:document>")
	out, err := Compose(context.Background(), packageBytes(t, base), packageBytes(t, closing))
	if err != nil {
		t.Fatal(err)
	}
	p, _ := opc.Read(context.Background(), out)
	xml, err := p.XML("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range opc.Elements(xml.Root(), opc.Word, "t") {
		if e.Text() == "Local namespace" {
			found = true
		}
	}
	if !found {
		t.Fatal("body-local namespace lost")
	}
}
func TestSplitReferenceFieldsAndPictureBulletsRejected(t *testing.T) {
	base := blankPackage(t, "Main")
	for _, kind := range []string{"field", "picture"} {
		closing := blankPackage(t, "Closing")
		if kind == "field" {
			xml, _ := closing.XML("word/document.xml")
			body := opc.Child(xml.Root(), opc.Word, "body")
			p := etree.NewElement("w:p")
			p.CreateElement("w:r").CreateElement("w:fldChar").CreateAttr("w:fldCharType", "begin")
			p.CreateElement("w:r").CreateElement("w:instrText").SetText(" RE")
			p.CreateElement("w:r").CreateElement("w:instrText").SetText("F scope ")
			p.CreateElement("w:r").CreateElement("w:fldChar").CreateAttr("w:fldCharType", "end")
			body.InsertChildAt(0, p)
			if err := closing.SetXML("word/document.xml", xml); err != nil {
				t.Fatal(err)
			}
		} else {
			n := opc.NewXML("numbering")
			n.Root().CreateElement("w:numPicBullet").CreateAttr("w:numPicBulletId", "1")
			if err := closing.SetXML("word/numbering.xml", n); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := Compose(context.Background(), packageBytes(t, base), packageBytes(t, closing)); err == nil {
			t.Fatal(kind)
		}
	}
}
func TestDefaultsMergeUsesSchemaOrderAndInheritedNamespaces(t *testing.T) {
	d := etree.NewDocument()
	if err := d.ReadFromString("<w:styles xmlns:w=\"" + opc.Word + "\"><w:docDefaults xmlns:a=\"" + opc.Word + "\"><a:pPrDefault><a:pPr><a:spacing a:after=\"80\"/></a:pPr></a:pPrDefault></w:docDefaults><w:style><w:rPr><w:b/></w:rPr></w:style></w:styles>"); err != nil {
		t.Fatal(err)
	}
	defaults := opc.Child(d.Root(), opc.Word, "docDefaults")
	style := opc.Child(d.Root(), opc.Word, "style")
	mergeDefaults(style, defaults)
	children := style.ChildElements()
	if len(children) != 2 || children[0].Tag != "pPr" || children[1].Tag != "rPr" {
		t.Fatal("style property order")
	}
	spacing := opc.Child(children[0], opc.Word, "spacing")
	if spacing == nil || opc.Attr(spacing, opc.Word, "after") != "80" {
		t.Fatal("default namespace lost")
	}
	paragraph := children[0]
	jc := paragraph.CreateElement("w:jc")
	jc.CreateAttr("w:val", "left")
	paragraph.RemoveChild(spacing)
	mergeDefaults(style, defaults)
	if children := paragraph.ChildElements(); children[0].Tag != "spacing" || children[1].Tag != "jc" {
		t.Fatal("paragraph property order")
	}
}
func TestEmbeddedClosingFontHasReachableFontTable(t *testing.T) {
	main := blankPackage(t, "Main")
	delete(main.Parts, "word/fontTable.xml")
	rel, _ := main.Relationships("word/document.xml")
	for _, r := range rel.Root().ChildElements() {
		if strings.HasSuffix(r.SelectAttrValue("Type", ""), "/fontTable") {
			rel.Root().RemoveChild(r)
		}
	}
	_ = main.SetXML(opc.RelName("word/document.xml"), rel)
	types, _ := main.XML("[Content_Types].xml")
	for _, e := range types.Root().ChildElements() {
		if e.SelectAttrValue("PartName", "") == "/word/fontTable.xml" {
			types.Root().RemoveChild(e)
		}
	}
	_ = main.SetXML("[Content_Types].xml", types)
	fake := make([]byte, 64)
	copy(fake, []byte{0, 1, 0, 0})
	binary.BigEndian.PutUint16(fake[4:6], 1)
	copy(fake[12:16], "OS/2")
	binary.BigEndian.PutUint32(fake[20:24], 32)
	binary.BigEndian.PutUint32(fake[24:28], 32)
	closing, err := fonts.Embed(context.Background(), packageBytes(t, blankPackage(t, "Closing")), []fonts.Face{{Name: "Closing face", Data: fake}})
	if err != nil {
		t.Fatal(err)
	}
	out, err := Compose(context.Background(), packageBytes(t, main), closing)
	if err != nil {
		t.Fatal(err)
	}
	merged, _ := opc.Read(context.Background(), out)
	relationships, _ := merged.Relationships("word/document.xml")
	found := false
	for _, r := range relationships.Root().ChildElements() {
		if strings.HasSuffix(r.SelectAttrValue("Type", ""), "/fontTable") {
			target, err := opc.Resolve("word/document.xml", r.SelectAttrValue("Target", ""))
			if err != nil || len(merged.Parts[target]) == 0 {
				t.Fatal("unreachable font table")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("font table relation missing")
	}
}
func TestStrictTemplatesRejectExcludedAndFieldAdjacentText(t *testing.T) {
	for _, fragmented := range []bool{false, true} {
		p := blankPackage(t, "Closing")
		xml, _ := p.XML("word/document.xml")
		paragraph := opc.Elements(xml.Root(), opc.Word, "p")[0]
		for _, child := range paragraph.ChildElements() {
			paragraph.RemoveChild(child)
		}
		if fragmented {
			paragraph.CreateElement("w:r").CreateElement("w:instrText").SetText("{")
			paragraph.CreateElement("w:r").CreateElement("w:instrText").SetText("{NAME}}")
		} else {
			paragraph.CreateElement("w:r").CreateElement("w:delText").SetText("{{NAME}}")
		}
		_ = p.SetXML("word/document.xml", xml)
		if _, err := Render(context.Background(), packageBytes(t, p), map[string]string{"NAME": "Value"}, Options{Document: report.Defaults()}); err == nil {
			t.Fatal("excluded placeholder accepted", fragmented)
		}
	}
	d, _ := editing.New(report.Defaults())
	p, _ := d.Paragraph("", editing.ParagraphOptions{})
	r, _ := p.Native().AddRun()
	_ = r.AddField(docx.NewPageNumberField())
	_ = p.Append("{{NAME}}", editing.TextStyle{})
	var input bytes.Buffer
	// Use the supported upstream writer for this admitted template fixture: the
	// editor deliberately rejects helper styling after fields rather than misbinding it.
	if _, err := d.Native().WriteTo(&input); err != nil {
		t.Fatal(err)
	}
	if _, err := Render(context.Background(), input.Bytes(), map[string]string{"NAME": "Value"}, Options{Document: report.Defaults()}); err == nil {
		t.Fatal("field-adjacent layout silently accepted")
	}
}
func TestMoreThanTenLiteralTokensDoNotOverlap(t *testing.T) {
	d, _ := editing.New(report.Defaults())
	values := map[string]string{}
	for i := 0; i < 15; i++ {
		key := fmt.Sprintf("KEY_%d", i)
		values[key] = fmt.Sprintf("Value_%d", i)
		_, _ = d.Paragraph("{{"+key+"}}", editing.ParagraphOptions{})
	}
	var input bytes.Buffer
	_ = d.Write(context.Background(), &input)
	if _, err := Render(context.Background(), input.Bytes(), values, Options{Document: report.Defaults()}); err != nil {
		t.Fatal(err)
	}
}

func TestRepeatedComposeWithCustomStylesPart(t *testing.T) {
	main := blankPackage(t, "Main")
	main.Parts["word/customStyles.xml"] = main.Parts["word/styles.xml"]
	delete(main.Parts, "word/styles.xml")
	relations, _ := main.Relationships("word/document.xml")
	for _, r := range relations.Root().ChildElements() {
		if strings.HasSuffix(r.SelectAttrValue("Type", ""), "/styles") {
			r.CreateAttr("Target", "customStyles.xml")
		}
	}
	if err := main.SetXML(opc.RelName("word/document.xml"), relations); err != nil {
		t.Fatal(err)
	}
	if err := main.ContentType("word/customStyles.xml", "application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"); err != nil {
		t.Fatal(err)
	}
	result := packageBytes(t, main)
	closing := packageBytes(t, blankPackage(t, "Closing"))
	for range 2 {
		var err error
		result, err = Compose(context.Background(), result, closing)
		if err != nil {
			t.Fatal(err)
		}
	}
	output, _ := opc.Read(context.Background(), result)
	styles, err := output.XML("word/customStyles.xml")
	if err != nil {
		t.Fatal(err)
	}
	used := map[string]bool{}
	for _, style := range opc.Elements(styles.Root(), opc.Word, "style") {
		id := opc.Attr(style, opc.Word, "styleId")
		if used[id] {
			t.Fatal("duplicate composed style", id)
		}
		used[id] = true
	}
	if !used["Closing1_Normal"] || !used["Closing2_Normal"] {
		t.Fatal("distinct closing styles missing")
	}
}
