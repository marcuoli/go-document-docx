package opc

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestReadRejectsDuplicateAndTraversal(t *testing.T) {
	build := func(t *testing.T, extra string) []byte {
		t.Helper()
		var b bytes.Buffer
		z := zip.NewWriter(&b)
		names := []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml"}
		if extra != "" {
			names = append(names, extra)
		}
		for _, name := range names {
			f, err := z.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.Write([]byte("<x/>")); err != nil {
				t.Fatal(err)
			}
		}
		if err := z.Close(); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	if _, err := Read(t.Context(), build(t, "")); err != nil {
		t.Fatal("positive control rejected", err)
	}
	for _, tc := range []struct{ name, extra, want string }{
		{"traversal", "../escape", "invalid package part name"},
		{"duplicate", "word/document.xml", "duplicate package part"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Read(t.Context(), build(t, tc.extra)); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatal("wrong admission result", err)
			}
		})
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
