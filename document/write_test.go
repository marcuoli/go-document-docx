package document_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
)

func TestCellTextColorsPreserveLiteralTextAndOwnedInputs(t *testing.T) {
	d := fixture(t)
	table := document.Table{Headers: []string{"Item", "Result"}, Widths: []int{4000, 5746}, Rows: [][]document.Cell{
		{{Text: "literal <value> & exact 9007199254740993"}, {Text: "✓", TextColor: "2E7D32"}},
		{{Text: "unavailable"}, {Text: "N/A", Tone: document.Unavailable, TextColor: "b3261e"}},
	}}
	d.AddTable(table)
	table.Rows[0][1].TextColor = "0000FF"
	var out bytes.Buffer
	if err := d.Write(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	main := parts(t, out.Bytes())["word/document.xml"]
	for _, expected := range []string{`<w:rPr><w:color w:val="2E7D32"/></w:rPr>`, `<w:color w:val="b3261e"/>`, "✓", "9007199254740993", "&lt;value&gt; &amp;", `w:fill="EDEBE8"`} {
		if !strings.Contains(main, expected) {
			t.Fatalf("missing %q", expected)
		}
	}
	if strings.Count(main, "<w:color") != 2 || strings.Contains(main, "0000FF") {
		t.Fatal("color leaked into inherited cells or caller mutation changed snapshot")
	}
}

func TestInvalidCellTextColorWritesNothing(t *testing.T) {
	for _, color := range []string{"#FF0000", "FFF", "GG0000", " FF0000", "FF0000 ", "FF000000", `FF0000"/><w:r>`} {
		t.Run(color, func(t *testing.T) {
			d := fixture(t)
			d.AddTable(document.Table{Headers: []string{"Result"}, Widths: []int{9746}, Rows: [][]document.Cell{{{Text: "value", TextColor: color}}}})
			var out bytes.Buffer
			if err := d.Write(context.Background(), &out); err == nil || out.Len() != 0 {
				t.Fatalf("invalid color was published: %v (%d bytes)", err, out.Len())
			}
		})
	}
}

func TestCellBoldColorUsesOrderedPropertiesAndLiteralText(t *testing.T) {
	d := fixture(t)
	table := document.Table{Headers: []string{"Item", "Value"}, Widths: []int{4000, 5746}, Rows: [][]document.Cell{{{Text: "review"}, {Text: "<Manual Entry>", TextColor: "B3261E", Bold: true}}, {{Text: "emphasis"}, {Text: "literal", Bold: true}}}}
	d.AddTable(table)
	table.Rows[0][1].Bold = false
	var out bytes.Buffer
	if err := d.Write(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	main := parts(t, out.Bytes())["word/document.xml"]
	if !strings.Contains(main, `<w:rPr><w:b/><w:color w:val="B3261E"/></w:rPr>`) || !strings.Contains(main, `&lt;Manual Entry&gt;`) || strings.Count(main, `<w:b/>`) != 2 {
		t.Fatal("bold/color property order, literal text or owned emphasis lost", main)
	}
}

func TestParagraphWithNextUsesBodyStyleAndPaginationHints(t *testing.T) {
	d := fixture(t)
	d.ParagraphWithNext("Brief purpose before the data")
	d.AddTable(document.Table{Headers: []string{"Value"}, Widths: []int{9746}, Rows: [][]document.Cell{{{Text: "literal"}}}})
	var out bytes.Buffer
	if err := d.Write(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	p := parts(t, out.Bytes())
	if !strings.Contains(p["word/document.xml"], `<w:pStyle w:val="Lead"/>`) || !strings.Contains(p["word/styles.xml"], `<w:pPr><w:keepNext/><w:keepLines/></w:pPr>`) {
		t.Fatal("purpose paragraph lacks pagination hints")
	}
}
