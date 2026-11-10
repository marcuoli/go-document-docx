package template

import (
	"context"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/internal/opc"
)

func TestTransitiveRelationshipNamespaceIsRetained(t *testing.T) {
	source := blankPackage(t, "Source")
	dest := blankPackage(t, "Main")
	source.Parts["word/custom/resource.xml"] = []byte("<item/>")
	source.Parts["word/media/picture.bin"] = []byte("synthetic graph resource")
	source.Parts["word/custom/_rels/resource.xml.rels"] = []byte("<p:Relationships xmlns:p=\"" + opc.Relations + "\"><p:Relationship Id=\"rId1\" Type=\"" + opc.Office + "/image\" Target=\"../media/picture.bin\"/></p:Relationships>")
	if err := source.ContentType("word/custom/resource.xml", "application/xml"); err != nil {
		t.Fatal(err)
	}
	if err := source.ContentType("word/media/picture.bin", "application/octet-stream"); err != nil {
		t.Fatal(err)
	}
	merger := &composer{ctx: context.Background(), source: source, dest: dest, prefix: "Closing1_", imported: map[string]string{}}
	imported, err := merger.part("word/custom/resource.xml")
	if err != nil {
		t.Fatal(err)
	}
	links, err := dest.Relationships(imported)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range links.Root().ChildElements() {
		if e.NamespaceURI() != opc.Relations {
			t.Fatal("relationship namespace lost")
		}
		target, err := opc.Resolve(imported, e.SelectAttrValue("Target", ""))
		if err != nil || len(dest.Parts[target]) == 0 {
			t.Fatal("missing dependency")
		}
	}
	if !strings.Contains(string(dest.Parts[opc.RelName(imported)]), "xmlns:p") {
		t.Fatal("ancestor declaration not retained")
	}
}
