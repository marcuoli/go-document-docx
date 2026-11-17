package document_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/document"
	doctemplate "github.com/marcuoli/go-document-docx/template"
)

func TestTableFontSizeStaysLocalAndSurvivesComposition(t *testing.T) {
	d := fixture(t)
	compact := document.Table{
		FontSize: 14, Headers: []string{"Identity", "Value"}, Widths: []int{2500, 2500},
		Rows: [][]document.Cell{{
			{Text: "Literal directory", TextColor: "B3261E", Alignment: "right"},
			{Runs: []document.CellRun{{Text: "1:value\n", TextColor: "007F00"}, {Text: "2:other"}}, Bold: true},
		}},
	}
	d.AddTable(compact)
	compact.FontSize = 72
	d.AddTable(document.Table{Headers: []string{"Default"}, Widths: []int{5000}, Rows: [][]document.Cell{{{Text: "Inherited"}}}})
	d.CodeTable("SQL", "select 1\nfrom dual")
	var out bytes.Buffer
	if err := d.Write(t.Context(), &out); err != nil {
		t.Fatal(err)
	}
	closing := fixture(t)
	closing.Paragraph("Closing prose")
	var tail bytes.Buffer
	if err := closing.Write(t.Context(), &tail); err != nil {
		t.Fatal(err)
	}
	combined, err := doctemplate.Compose(t.Context(), out.Bytes(), tail.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"literal": out.Bytes(), "composed": combined} {
		t.Run(name, func(t *testing.T) {
			p := parts(t, data)
			xml := etree.NewDocument()
			if err := xml.ReadFromString(p["word/document.xml"]); err != nil {
				t.Fatal(err)
			}
			tables := xml.FindElements("//w:tbl")
			if len(tables) != 3 {
				t.Fatal("tables lost", len(tables))
			}
			for index, table := range tables {
				runs := table.FindElements(".//w:r")
				if len(runs) == 0 {
					t.Fatal("text runs lost")
				}
				for _, run := range runs {
					size, complex := run.FindElement("w:rPr/w:sz"), run.FindElement("w:rPr/w:szCs")
					if index != 0 {
						if size != nil || complex != nil {
							t.Fatal("size leaked into another table")
						}
						continue
					}
					if size == nil || complex == nil || size.SelectAttrValue("w:val", "") != "14" || complex.SelectAttrValue("w:val", "") != "14" {
						t.Fatal("header/body/rich run size not preserved", run)
					}
				}
			}
			first := tables[0]
			red := first.FindElement("w:tr[2]/w:tc[1]/w:p/w:r/w:rPr/w:color")
			if red == nil || red.SelectAttrValue("w:val", "") != "B3261E" {
				t.Fatal("literal color lost")
			}
			align := first.FindElement("w:tr[2]/w:tc[1]/w:p/w:pPr/w:jc")
			if align == nil || align.SelectAttrValue("w:val", "") != "right" {
				t.Fatal("alignment lost")
			}
			rich := first.FindElements("w:tr[2]/w:tc[2]/w:p/w:r")
			if len(rich) != 2 || rich[0].FindElement("w:rPr/w:b") == nil || rich[1].FindElement("w:rPr/w:b") == nil {
				t.Fatal("rich spans or emphasis lost")
			}
			if color := rich[0].FindElement("w:rPr/w:color"); color == nil || color.SelectAttrValue("w:val", "") != "007F00" {
				t.Fatal("span color lost")
			}
			if !strings.Contains(p["word/document.xml"], "Literal directory") || strings.Contains(p["word/document.xml"], `w:val="72"`) {
				t.Fatal("owned literal/size changed")
			}
		})
	}
}

func TestTableFontSizeValidationPrecedesOutput(t *testing.T) {
	for _, size := range []int{-1, 11, 145, int(^uint(0) >> 1)} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			d := fixture(t)
			d.Paragraph("Content before the invalid table")
			d.AddTable(document.Table{FontSize: size, Headers: []string{"Value"}, Widths: []int{5000}})
			var out bytes.Buffer
			if err := d.Write(t.Context(), &out); err == nil || out.Len() != 0 {
				t.Fatal("invalid size wrote output", err, out.Len())
			}
		})
	}
	for _, size := range []int{0, 12, 144} {
		d := fixture(t)
		d.AddTable(document.Table{FontSize: size, Headers: []string{"Empty table"}, Widths: []int{5000}})
		var out bytes.Buffer
		if err := d.Write(t.Context(), &out); err != nil {
			t.Fatal(size, err)
		}
	}
}
