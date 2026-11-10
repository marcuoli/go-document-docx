package fonts

import (
	"bytes"
	"context"
	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/report"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/internal/opc"
)

func TestExistingFontTableTargetIsUsed(t *testing.T) {
	original := baseDocument(t)
	p, err := opc.Read(context.Background(), original)
	if err != nil {
		t.Fatal(err)
	}
	const custom = "word/resources/typefaces.xml"
	p.Parts[custom] = p.Parts["word/fontTable.xml"]
	delete(p.Parts, "word/fontTable.xml")
	rel, err := p.Relationships("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rel.Root().ChildElements() {
		if strings.HasSuffix(r.SelectAttrValue("Type", ""), "/fontTable") {
			r.CreateAttr("Target", "resources/typefaces.xml")
		}
	}
	if err := p.SetXML(opc.RelName("word/document.xml"), rel); err != nil {
		t.Fatal(err)
	}
	types, _ := p.XML("[Content_Types].xml")
	for _, e := range types.Root().ChildElements() {
		if e.SelectAttrValue("PartName", "") == "/word/fontTable.xml" {
			e.CreateAttr("PartName", "/"+custom)
		}
	}
	if err := p.SetXML("[Content_Types].xml", types); err != nil {
		t.Fatal(err)
	}
	input := packageDocument(t, p)
	output, err := Embed(context.Background(), input, []Face{{Name: "Synthetic", Data: fixtureFont(0)}})
	if err != nil {
		t.Fatal(err)
	}
	out, _ := opc.Read(context.Background(), output)
	if _, exists := out.Parts["word/fontTable.xml"]; exists {
		t.Fatal("created unreachable font table")
	}
	fonts, err := out.XML(custom)
	if err != nil {
		t.Fatal(err)
	}
	if len(opc.Elements(fonts.Root(), opc.Word, "embedRegular")) != 1 {
		t.Fatal("font not embedded into related table")
	}
	links, err := out.Relationships(custom)
	if err != nil {
		t.Fatal(err)
	}
	for _, link := range links.Root().ChildElements() {
		target, err := opc.Resolve(custom, link.SelectAttrValue("Target", ""))
		if err != nil || len(out.Parts[target]) == 0 {
			t.Fatal("unreachable font data", target, err)
		}
	}
	if _, err := Embed(context.Background(), input, []Face{{Name: "Bad\x01Name", Data: fixtureFont(0)}}); err == nil {
		t.Fatal("invalid family accepted")
	}
}

func baseDocument(t *testing.T) []byte {
	t.Helper()
	d, _ := document.New(report.Defaults())
	d.Paragraph("Synthetic font package")
	var b bytes.Buffer
	if err := d.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func packageDocument(t *testing.T, p *opc.Package) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := p.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
