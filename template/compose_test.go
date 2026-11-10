package template

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/internal/opc"
	"github.com/marcuoli/go-document-docx/report"
)

func examplePage(t *testing.T, title string, c color.RGBA) []byte {
	t.Helper()
	d, err := editing.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Paragraph(title, editing.ParagraphOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Bookmark("scope"); err != nil {
		t.Fatal(err)
	}
	if err := p.Link("Scope", "scope", "Go to scope", true); err != nil {
		t.Fatal(err)
	}
	pixels := image.NewRGBA(image.Rect(0, 0, 4, 4))
	pixels.Set(0, 0, c)
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, pixels); err != nil {
		t.Fatal(err)
	}
	if err := p.Image(imageBytes.Bytes(), 720, 720); err != nil {
		t.Fatal(err)
	}
	if _, err := d.List("Action "+title, true, 0); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := d.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestClosingPageRemapsResourcesListsAndBookmarks(t *testing.T) {
	first := examplePage(t, "Main review", color.RGBA{R: 255, A: 255})
	last := examplePage(t, "Closing review", color.RGBA{B: 255, A: 255})
	original := bytes.Clone(first)
	result, err := Compose(context.Background(), first, last)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, original) {
		t.Fatal("main input changed")
	}
	p, err := opc.Read(context.Background(), result)
	if err != nil {
		t.Fatal(err)
	}
	xml, err := p.XML("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(p.Parts["word/document.xml"]), "Closing review") {
		t.Fatal("closing body missing")
	}
	names := map[string]bool{}
	for _, e := range opc.Elements(xml.Root(), opc.Word, "bookmarkStart") {
		name := opc.Attr(e, opc.Word, "name")
		if names[name] {
			t.Fatal("duplicate bookmark", name)
		}
		names[name] = true
	}
	if !names["scope"] || !names["Closing1_scope"] {
		t.Fatal(names)
	}
	rel, err := p.Relationships("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	images := 0
	for _, e := range rel.Root().ChildElements() {
		if strings.HasSuffix(e.SelectAttrValue("Type", ""), "/image") {
			images++
			target, err := opc.Resolve("word/document.xml", e.SelectAttrValue("Target", ""))
			if err != nil || len(p.Parts[target]) == 0 {
				t.Fatal(target, err)
			}
		}
	}
	if images != 2 {
		t.Fatal("image relationship collision", images)
	}
	number, err := p.XML("word/numbering.xml")
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, e := range opc.Elements(number.Root(), opc.Word, "num") {
		id := opc.Attr(e, opc.Word, "numId")
		if ids[id] {
			t.Fatal("duplicate numbering instance")
		}
		ids[id] = true
	}
	if len(ids) != 4 {
		t.Fatal("lists were not remapped", len(ids))
	}
	if _, err := Compose(context.Background(), result, last); err != nil {
		t.Fatal("second append failed", err)
	}
}
func TestTemplateImageAndUnsupportedClosingConstruct(t *testing.T) {
	d, _ := editing.New(report.Defaults())
	_, _ = d.Paragraph("{{IMAGE}}", editing.ParagraphOptions{})
	var source bytes.Buffer
	if err := d.Write(context.Background(), &source); err != nil {
		t.Fatal(err)
	}
	imageData := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var imageBytes bytes.Buffer
	_ = png.Encode(&imageBytes, imageData)
	rendered, err := Render(context.Background(), source.Bytes(), nil, Options{Document: report.Defaults(), Images: map[string]Image{"IMAGE": {Data: imageBytes.Bytes(), Width: 720}}})
	if err != nil {
		t.Fatal(err)
	}
	p, _ := opc.Read(context.Background(), rendered)
	if !strings.Contains(string(p.Parts["word/document.xml"]), "drawing") {
		t.Fatal("image absent")
	}
	bad, _ := opc.Read(context.Background(), source.Bytes())
	xml, _ := bad.XML("word/document.xml")
	body := opc.Child(xml.Root(), opc.Word, "body")
	body.CreateElement("w:footnoteReference").CreateAttr("w:id", "1")
	_ = bad.SetXML("word/document.xml", xml)
	var b bytes.Buffer
	_ = bad.Write(context.Background(), &b)
	if _, err := Compose(context.Background(), source.Bytes(), b.Bytes()); err == nil {
		t.Fatal("unsupported notes silently imported")
	}
}
