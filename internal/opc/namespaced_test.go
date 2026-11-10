package opc

import (
	"testing"

	"github.com/beevik/etree"
)

func TestPrefixedPackageListsGetQualifiedChildren(t *testing.T) {
	rel := etree.NewDocument()
	if err := rel.ReadFromString("<p:Relationships xmlns:p=\"" + Relations + "\"/>"); err != nil {
		t.Fatal(err)
	}
	AddRelation(rel, "hyperlink", "https://example.com", "External")
	if rel.Root().ChildElements()[0].NamespaceURI() != Relations {
		t.Fatal("new relationship has no package namespace")
	}
	p := &Package{Parts: map[string][]byte{"[Content_Types].xml": []byte("<c:Types xmlns:c=\"" + Types + "\"/>")}}
	if err := p.ContentType("word/media/example.bin", "application/octet-stream"); err != nil {
		t.Fatal(err)
	}
	types, err := p.XML("[Content_Types].xml")
	if err != nil {
		t.Fatal(err)
	}
	if types.Root().ChildElements()[0].NamespaceURI() != Types {
		t.Fatal("new override has no content-type namespace")
	}
}
func TestXMLAdmissionRejectsDeepOrUnboundTrees(t *testing.T) {
	p := &Package{Parts: map[string][]byte{"x": []byte("<w:p/>")}}
	if _, err := p.XML("x"); err == nil {
		t.Fatal("unbound prefix accepted")
	}
	text := ""
	for range 129 {
		text += "<x>"
	}
	for range 129 {
		text += "</x>"
	}
	p.Parts["x"] = []byte(text)
	if _, err := p.XML("x"); err == nil {
		t.Fatal("unbounded nesting accepted")
	}
}

func TestRootPartRelativeTargets(t *testing.T) {
	for _, sample := range []struct{ from, to, want string }{{"fonts.xml", "word/fonts/font1.odttf", "word/fonts/font1.odttf"}, {"word/document.xml", "fonts.xml", "../fonts.xml"}, {"word/fonts.xml", "word/fonts/font1.odttf", "fonts/font1.odttf"}} {
		if got := Relative(sample.from, sample.to); got != sample.want {
			t.Fatal(got, sample.want)
		}
	}
	p := &Package{Parts: map[string][]byte{"x": []byte("<x/><y/>")}}
	if _, err := p.XML("x"); err == nil {
		t.Fatal("multiple roots accepted")
	}
}
