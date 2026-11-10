package opc

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"
)

func TestReadRejectsDuplicateAndTraversal(t *testing.T) {
	for _, name := range []string{"../escape", "word/document.xml"} {
		var b bytes.Buffer
		z := zip.NewWriter(&b)
		for _, n := range []string{"word/document.xml", name} {
			f, _ := z.Create(n)
			_, _ = f.Write([]byte("<x/>"))
		}
		_ = z.Close()
		if _, err := Read(context.Background(), b.Bytes()); err == nil {
			t.Fatal(name)
		}
	}
}
func TestNamespaceHelpersAndEntityRejection(t *testing.T) {
	p := &Package{Parts: map[string][]byte{"x": []byte("<x:p xmlns:x=\"" + Word + "\"><x:r x:val=\"literal\"/></x:p>")}}
	d, err := p.XML("x")
	if err != nil {
		t.Fatal(err)
	}
	r := Child(d.Root(), Word, "r")
	if r == nil || Attr(r, Word, "val") != "literal" {
		t.Fatal("namespace alias lost")
	}
	p.Parts["x"] = []byte("<!DOCTYPE x [<!ENTITY a 'bad'>]><x/>")
	if _, err := p.XML("x"); err == nil {
		t.Fatal("DTD accepted")
	}
}
