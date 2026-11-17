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

func TestComposeRetainsPartialDefaultAttributes(t *testing.T) {
	main, closing := blankPackage(t, "Main"), blankPackage(t, "Closing")
	styles, err := closing.XML("word/styles.xml")
	if err != nil {
		t.Fatal(err)
	}
	defaults := opc.Child(styles.Root(), opc.Word, "docDefaults")
	ppr := opc.Child(opc.Child(defaults, opc.Word, "pPrDefault"), opc.Word, "pPr")
	opc.Ensure(ppr, "spacing").CreateAttr("w:before", "240")
	rpr := opc.Child(opc.Child(defaults, opc.Word, "rPrDefault"), opc.Word, "rPr")
	fonts := opc.Ensure(rpr, "rFonts")
	fonts.CreateAttr("w:ascii", "DefaultLatin")
	fonts.CreateAttr("w:hAnsi", "DefaultWestern")
	fonts.CreateAttr("w:eastAsia", "DefaultAsian")
	for _, style := range opc.Elements(styles.Root(), opc.Word, "style") {
		if opc.Attr(style, opc.Word, "styleId") == "Normal" {
			opc.Properties(style, "pPr").CreateElement("w:spacing").CreateAttr("w:after", "80")
			opc.Ordered(style, "rPr", opc.StyleOrder).CreateElement("w:rFonts").CreateAttr("w:ascii", "OverrideLatin")
		}
	}
	if err := closing.SetXML("word/styles.xml", styles); err != nil {
		t.Fatal(err)
	}
	out, err := Compose(t.Context(), packageBytes(t, main), packageBytes(t, closing))
	if err != nil {
		t.Fatal(err)
	}
	got, err := opc.Read(t.Context(), out)
	if err != nil {
		t.Fatal(err)
	}
	result, err := got.XML("word/styles.xml")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range opc.Elements(result.Root(), opc.Word, "style") {
		if opc.Attr(style, opc.Word, "styleId") != "Closing1_Normal" {
			continue
		}
		spacing := opc.Child(opc.Child(style, opc.Word, "pPr"), opc.Word, "spacing")
		if opc.Attr(spacing, opc.Word, "before") != "240" || opc.Attr(spacing, opc.Word, "after") != "80" {
			t.Fatal("partial spacing defaults lost")
		}
		family := opc.Child(opc.Child(style, opc.Word, "rPr"), opc.Word, "rFonts")
		for key, want := range map[string]string{"ascii": "OverrideLatin", "hAnsi": "DefaultWestern", "eastAsia": "DefaultAsian"} {
			if opc.Attr(family, opc.Word, key) != want {
				t.Fatal("partial font defaults lost", key)
			}
		}
		return
	}
	t.Fatal("imported default style missing")
}

func TestDefaultFontSlotsKeepExplicitLiteralAndThemeChoices(t *testing.T) {
	for _, tc := range []struct{ name, explicitKey, explicitValue, defaultKey, defaultValue string }{
		{"literal replaces theme", "ascii", "ExplicitLatin", "asciiTheme", "minorHAnsi"},
		{"theme replaces literal", "asciiTheme", "majorHAnsi", "ascii", "DefaultLatin"},
		{"complex script theme", "cstheme", "majorBidi", "cs", "DefaultComplex"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := etree.NewElement("w:rFonts")
			current.CreateAttr("xmlns:w", opc.Word)
			current.CreateAttr("w:"+tc.explicitKey, tc.explicitValue)
			defaults := etree.NewElement("w:rFonts")
			defaults.CreateAttr("xmlns:w", opc.Word)
			defaults.CreateAttr("w:"+tc.defaultKey, tc.defaultValue)
			defaults.CreateAttr("w:eastAsia", "InheritedAsian")
			mergeDefaultAttributes(current, defaults)
			if opc.Attr(current, opc.Word, tc.explicitKey) != tc.explicitValue || opc.Attr(current, opc.Word, tc.defaultKey) != "" || opc.Attr(current, opc.Word, "eastAsia") != "InheritedAsian" {
				t.Fatal("font slot inheritance changed")
			}
		})
	}
}

func TestExplicitIndentationOverridesInheritedAlternatives(t *testing.T) {
	for _, tc := range []struct{ name, currentKey, currentValue, defaultKey, defaultValue string }{
		{"first line replaces hanging", "firstLine", "180", "hanging", "360"},
		{"hanging replaces first line", "hanging", "360", "firstLine", "180"},
		{"twips replace character units", "left", "720", "leftChars", "200"},
		{"character units replace twips", "rightChars", "200", "right", "720"},
		{"logical start replaces physical left", "start", "720", "left", "360"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := etree.NewElement("w:ind")
			current.CreateAttr("xmlns:w", opc.Word)
			current.CreateAttr("w:"+tc.currentKey, tc.currentValue)
			defaults := etree.NewElement("w:ind")
			defaults.CreateAttr("xmlns:w", opc.Word)
			defaults.CreateAttr("w:"+tc.defaultKey, tc.defaultValue)
			mergeDefaultAttributes(current, defaults)
			if opc.Attr(current, opc.Word, tc.currentKey) != tc.currentValue || opc.Attr(current, opc.Word, tc.defaultKey) != "" {
				t.Fatal("inherited alternative overrides explicit indentation")
			}
		})
	}
}
