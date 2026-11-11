package document_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
)

func TestStyleRunOrderAndTableShading(t *testing.T) {
	d := fixture(t)
	d.Heading("Review", 1)
	d.AddTable(document.Table{Headers: []string{"Value"}, Widths: []int{2000}, Rows: [][]document.Cell{{{Text: "exact"}}}})
	var b bytes.Buffer
	if err := d.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	p := parts(t, b.Bytes())
	decoder := xml.NewDecoder(bytes.NewBufferString(p["word/styles.xml"]))
	inRun, seenSize := false, false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		switch e := token.(type) {
		case xml.StartElement:
			if e.Name.Local == "rPr" {
				inRun = true
				seenSize = false
			}
			if inRun && e.Name.Local == "sz" {
				seenSize = true
			}
			if inRun && e.Name.Local == "b" && seenSize {
				t.Fatal("style bold occurs after size; Word requires CT_StyleRunProperties sequence")
			}
		case xml.EndElement:
			if e.Name.Local == "rPr" {
				inRun = false
			}
		}
	}
	decoder = xml.NewDecoder(bytes.NewBufferString(p["word/document.xml"]))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if e, ok := token.(xml.StartElement); ok && e.Name.Local == "shd" {
			val := ""
			for _, a := range e.Attr {
				if a.Name.Local == "val" {
					val = a.Value
				}
			}
			if val != "clear" {
				t.Fatal("solid table shading lacks the required clear pattern", val)
			}
		}
	}
}
