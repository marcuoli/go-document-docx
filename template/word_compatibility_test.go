package template

import (
	"bytes"
	"context"
	"image/color"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/internal/opc"
)

func TestComposeIsolatesStyleNamesAndRetainsSingleNoteParts(t *testing.T) {
	raw := examplePage(t, "Review", color.RGBA{A: 255})
	p, err := opc.Read(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	styles, _ := p.XML("word/styles.xml")
	normal := opc.Child(styles.Root(), opc.Word, "style")
	normal.CreateElement("w:aliases").CreateAttr("w:val", "Body, review")
	_ = p.SetXML("word/styles.xml", styles)
	rels, _ := p.Relationships("word/document.xml")
	for _, kind := range []string{"footnotes", "endnotes"} {
		name := "word/" + kind + ".xml"
		p.Parts[name] = []byte(`<w:` + kind + ` xmlns:w="` + opc.Word + `"/>`)
		_ = p.ContentType(name, "application/vnd.openxmlformats-officedocument.wordprocessingml."+kind+"+xml")
		opc.AddRelation(rels, kind, kind+".xml", "")
	}
	_ = p.SetXML(opc.RelName("word/document.xml"), rels)
	number, _ := p.XML("word/numbering.xml")
	number.Root().CreateElement("w:numIdMacAtCleanup").CreateAttr("w:val", "1")
	_ = p.SetXML("word/numbering.xml", number)
	var input bytes.Buffer
	if err := p.Write(context.Background(), &input); err != nil {
		t.Fatal(err)
	}
	result, err := Compose(context.Background(), input.Bytes(), input.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	out, err := opc.Read(context.Background(), result)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := out.XML("word/styles.xml")
	names := map[string]bool{}
	for _, style := range s.Root().ChildElements() {
		if style.Tag != "style" {
			continue
		}
		for _, tag := range []string{"name", "aliases"} {
			e := opc.Child(style, opc.Word, tag)
			if e == nil {
				continue
			}
			for _, name := range strings.Split(opc.Attr(e, opc.Word, "val"), ",") {
				key := strings.ToLower(strings.TrimSpace(name))
				if names[key] {
					t.Fatal("duplicate Word style name/alias", name)
				}
				names[key] = true
			}
		}
	}
	r, _ := out.Relationships("word/document.xml")
	for _, kind := range []string{"footnotes", "endnotes"} {
		count := 0
		for _, e := range r.Root().ChildElements() {
			if strings.HasSuffix(e.SelectAttrValue("Type", ""), "/"+kind) {
				count++
			}
		}
		if count != 1 {
			t.Fatal("duplicate singleton note relationship", kind, count)
		}
	}
	n, _ := out.XML("word/numbering.xml")
	seenTail := false
	for _, e := range n.Root().ChildElements() {
		if e.Tag == "numIdMacAtCleanup" {
			seenTail = true
		}
		if seenTail && e.Tag == "num" {
			t.Fatal("numbering instance follows the terminal cleanup identifier")
		}
	}
}

func TestComposeRejectsStyleDependentContentsFields(t *testing.T) {
	base := blankPackage(t, "Main")
	for _, complex := range []bool{false, true} {
		source := blankPackage(t, "Closing")
		xml, _ := source.XML("word/document.xml")
		body := opc.Child(xml.Root(), opc.Word, "body")
		p := body.CreateElement("w:p")
		if complex {
			p.CreateElement("w:r").CreateElement("w:fldChar").CreateAttr("w:fldCharType", "begin")
			p.CreateElement("w:r").CreateElement("w:instrText").SetText(" TO")
			p.CreateElement("w:r").CreateElement("w:instrText").SetText(`C \t "Custom Style,1" `)
			p.CreateElement("w:r").CreateElement("w:fldChar").CreateAttr("w:fldCharType", "end")
		} else {
			p.CreateElement("w:fldSimple").CreateAttr("w:instr", `TOC \t "Custom Style,1"`)
		}
		_ = source.SetXML("word/document.xml", xml)
		if _, err := Compose(context.Background(), packageBytes(t, base), packageBytes(t, source)); err == nil {
			t.Fatal("style-name field silently imported", complex)
		}
	}
}

func TestComposeNumberingWithOnlyCleanupIdentifier(t *testing.T) {
	base := blankPackage(t, "Main")
	number := opc.NewXML("numbering")
	number.Root().CreateElement("w:numIdMacAtCleanup").CreateAttr("w:val", "1")
	_ = base.SetXML("word/numbering.xml", number)
	source := examplePage(t, "List", color.RGBA{A: 255})
	result, err := Compose(context.Background(), packageBytes(t, base), source)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := opc.Read(context.Background(), result)
	n, _ := p.XML("word/numbering.xml")
	children := n.Root().ChildElements()
	if len(children) < 3 || children[len(children)-1].Tag != "numIdMacAtCleanup" {
		t.Fatal("cleanup identifier is no longer terminal", n.Root().FullTag())
	}
}

func TestComposeNormalizesStyleRunOrderWithoutChangingValues(t *testing.T) {
	base := blankPackage(t, "Main")
	// Model a source template that appended color/size after szCs and an extension.
	styles, _ := base.XML("word/styles.xml")
	style := opc.Child(styles.Root(), opc.Word, "style")
	if old := opc.Child(style, opc.Word, "rPr"); old != nil {
		style.RemoveChild(old)
	}
	pr := style.CreateElement("w:rPr")
	pr.CreateElement("w:szCs").CreateAttr("w:val", "21")
	extension := pr.CreateElement("w14:numSpacing")
	extension.CreateAttr("xmlns:w14", "http://schemas.microsoft.com/office/word/2010/wordml")
	extension.CreateAttr("w14:val", "tabular")
	pr.CreateElement("w:color").CreateAttr("w:val", "123456")
	pr.CreateElement("w:sz").CreateAttr("w:val", "19")
	_ = base.SetXML("word/styles.xml", styles)
	sourceID := opc.Attr(style, opc.Word, "styleId")
	input := packageBytes(t, base)
	result, err := Compose(context.Background(), input, input)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := opc.Read(context.Background(), result)
	s, _ := p.XML("word/styles.xml")
	checked := 0
	for _, style := range opc.Elements(s.Root(), opc.Word, "style") {
		id := opc.Attr(style, opc.Word, "styleId")
		if id != sourceID && id != "Closing1_"+sourceID {
			continue
		}
		checked++
		rpr := opc.Child(style, opc.Word, "rPr")
		if rpr == nil {
			t.Fatal("style properties disappeared", id)
		}

		positions := map[string]int{}
		for i, child := range rpr.ChildElements() {
			positions[child.Tag] = i
		}
		if positions["color"] >= positions["sz"] || positions["sz"] >= positions["szCs"] || positions["szCs"] >= positions["numSpacing"] {
			t.Fatal("style run schema order lost", positions)
		}
		extension := opc.Child(rpr, "http://schemas.microsoft.com/office/word/2010/wordml", "numSpacing")
		if extension == nil || opc.Attr(extension, "http://schemas.microsoft.com/office/word/2010/wordml", "val") != "tabular" {
			t.Fatal("extension value lost", id)
		}
		for tag, value := range map[string]string{"color": "123456", "sz": "19", "szCs": "21"} {
			if opc.Attr(opc.Child(rpr, opc.Word, tag), opc.Word, "val") != value {
				t.Fatal("property value changed", tag)
			}
		}
	}
	if checked != 2 {
		t.Fatal("expected source and imported style definitions", checked)
	}
	original, _ := base.XML("word/styles.xml")
	if opc.Child(opc.Child(original.Root(), opc.Word, "style"), opc.Word, "rPr").ChildElements()[0].Tag != "szCs" {
		t.Fatal("source package changed")
	}
}
