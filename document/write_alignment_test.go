package document_test

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
)

func TestCellAlignment(t *testing.T) {
	for _, alignment := range []string{"", "left", "center", "right", "justify"} {
		for _, rich := range []bool{false, true} {
			name := alignment
			if name == "" {
				name = "inherit"
			}
			if rich {
				name += "/runs"
			} else {
				name += "/text"
			}
			t.Run(name, func(t *testing.T) {
				d := fixture(t)
				cell := document.Cell{Text: "1,234.50\n0,0008", Alignment: alignment, Bold: true, TextColor: "007F00", Tone: document.Difference}
				if rich {
					cell.Text = ""
					cell.Runs = []document.CellRun{{Text: "1,234.50\n"}, {Text: "0,0008", TextColor: "B3261E"}}
				}
				rows := [][]document.Cell{{cell, {Text: "42"}}}
				d.AddTable(document.Table{Headers: []string{"Value", "Literal"}, Widths: []int{2500, 2500}, Rows: rows})
				rows[0][0].Alignment = "mutated"
				var out bytes.Buffer
				if err := d.Write(t.Context(), &out); err != nil {
					t.Fatal(err)
				}
				var parsed struct {
					Body struct {
						Table struct {
							Rows []struct {
								Cells []struct {
									Paragraph struct {
										Properties struct {
											Alignment *struct {
												Value string `xml:"val,attr"`
											} `xml:"jc"`
										} `xml:"pPr"`
										XML string `xml:",innerxml"`
									} `xml:"p"`
								} `xml:"tc"`
							} `xml:"tr"`
						} `xml:"tbl"`
					} `xml:"body"`
				}
				if err := xml.Unmarshal([]byte(parts(t, out.Bytes())["word/document.xml"]), &parsed); err != nil {
					t.Fatal(err)
				}
				body := parsed.Body.Table.Rows[1].Cells
				got := body[0].Paragraph.Properties.Alignment
				want := alignment
				if want == "justify" {
					want = "both"
				}
				if alignment == "" {
					if got != nil {
						t.Fatal("inherited cell acquired a direct alignment")
					}
				} else if got == nil || got.Value != want {
					t.Fatal("cell did not retain its explicit owned alignment", got)
				}
				if body[1].Paragraph.Properties.Alignment != nil || parsed.Body.Table.Rows[0].Cells[0].Paragraph.Properties.Alignment != nil {
					t.Fatal("alignment leaked to sibling or header")
				}
				content := body[0].Paragraph.XML
				if codeRunText(t, content) != "1,234.50\n0,0008" || !strings.Contains(content, `<w:b/>`) || !strings.Contains(content, `w:val="007F00"`) {
					t.Fatal("alignment changed literal text or emphasis", content)
				}
				if rich && !strings.Contains(content, `w:val="B3261E"`) {
					t.Fatal("alignment changed run color")
				}
			})
		}
	}
}

func TestCellAlignmentRejectsInvalidBeforeOutput(t *testing.T) {
	for _, alignment := range []string{"RIGHT", " right", "right ", "both", "start", `right"/><w:evil/>`} {
		t.Run(alignment, func(t *testing.T) {
			d := fixture(t)
			d.Paragraph("Valid earlier content must not reach the writer")
			d.AddTable(document.Table{Headers: []string{"Value"}, Widths: []int{5000}, Rows: [][]document.Cell{{{Alignment: alignment, Runs: []document.CellRun{{Text: "42"}}}}}})
			var out bytes.Buffer
			if err := d.Write(t.Context(), &out); err == nil || !strings.Contains(err.Error(), "alignment") || out.Len() != 0 {
				t.Fatalf("invalid alignment: error=%v output=%d", err, out.Len())
			}
		})
	}
}
