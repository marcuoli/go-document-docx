package fonts

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/internal/opc"
	"github.com/marcuoli/go-document-docx/report"
)

func fixtureFont(flags uint16) []byte {
	b := make([]byte, 64)
	copy(b, []byte{0, 1, 0, 0})
	binary.BigEndian.PutUint16(b[4:6], 1)
	copy(b[12:16], "OS/2")
	binary.BigEndian.PutUint32(b[20:24], 32)
	binary.BigEndian.PutUint32(b[24:28], 32)
	binary.BigEndian.PutUint16(b[40:42], flags)
	return b
}
func TestFontEmbeddingPackageAndReversibleKey(t *testing.T) {
	d, _ := document.New(report.Defaults())
	d.Paragraph("Font embedding package check")
	var input bytes.Buffer
	if err := d.Write(context.Background(), &input); err != nil {
		t.Fatal(err)
	}
	raw := fixtureFont(0)
	original := bytes.Clone(raw)
	result, err := Embed(context.Background(), input.Bytes(), []Face{{Name: "Synthetic face", Data: raw}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, original) {
		t.Fatal("font input changed")
	}
	p, err := opc.Read(context.Background(), result)
	if err != nil {
		t.Fatal(err)
	}
	table, _ := p.XML("word/fontTable.xml")
	embedded := opc.Elements(table.Root(), opc.Word, "embedRegular")
	if len(embedded) != 1 {
		t.Fatal("face absent")
	}
	key := strings.NewReplacer("{", "", "}", "", "-", "").Replace(opc.Attr(embedded[0], opc.Word, "fontKey"))
	guid, err := hex.DecodeString(key)
	if err != nil {
		t.Fatal(err)
	}
	encoded := bytes.Clone(p.Parts["word/fonts/font1.odttf"])
	for i := 0; i < 32; i++ {
		encoded[i] ^= guid[15-i%16]
	}
	if !bytes.Equal(encoded, raw) {
		t.Fatal("obfuscation key does not round-trip")
	}
	rel, _ := p.XML("word/_rels/fontTable.xml.rels")
	if !strings.Contains(string(p.Parts["[Content_Types].xml"]), "obfuscatedFont") || len(rel.Root().ChildElements()) != 1 {
		t.Fatal("font binding missing")
	}
}
func TestFontRestrictionsAndMalformedTables(t *testing.T) {
	for _, data := range [][]byte{fixtureFont(2), fixtureFont(0x200), []byte("ttcf"), make([]byte, 33)} {
		if err := embeddingAllowed(data); err == nil {
			t.Fatal("restricted/malformed font accepted")
		}
	}
	if err := embeddingAllowed(fixtureFont(0x100)); err != nil {
		t.Fatal("complete face incorrectly treated as subset", err)
	}
}

func TestEmbedPreservesLocalNamespaceBindings(t *testing.T) {
	p, err := opc.Read(t.Context(), baseDocument(t))
	if err != nil {
		t.Fatal(err)
	}
	p.Parts["word/fontTable.xml"] = []byte(`<w:fonts xmlns:w="` + opc.Word + `" xmlns:r="` + opc.Office + `"><a:font xmlns:a="` + opc.Word + `" xmlns:w="urn:foreign" xmlns:r="urn:foreign-office" a:name="Synthetic" w:name="foreign" r:id="retained"/></w:fonts>`)
	out, err := Embed(t.Context(), packageDocument(t, p), []Face{{Name: "Synthetic", Data: fixtureFont(0)}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := opc.Read(t.Context(), out)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := got.XML("word/fontTable.xml")
	if err != nil {
		t.Fatal(err)
	}
	slots := opc.Elements(doc.Root(), opc.Word, "embedRegular")
	if len(slots) != 1 || opc.Attr(slots[0], opc.Word, "fontKey") == "" || opc.Attr(slots[0], opc.Office, "id") == "" {
		t.Fatal("embedded face has incorrect expanded namespaces")
	}
	font := opc.Elements(doc.Root(), opc.Word, "font")[0]
	if opc.Attr(font, "urn:foreign", "name") != "foreign" || opc.Attr(font, "urn:foreign-office", "id") != "retained" {
		t.Fatal("foreign attributes rebound")
	}
}
