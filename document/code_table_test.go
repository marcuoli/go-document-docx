package document_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
)

func TestCodeTablePreservesLiteralTextAndPagination(t *testing.T) {
	d := fixture(t)
	code := "  SELECT '<&>' AS \"ação 東京 😀\";\r\n\t-- next\n\n" + strings.Repeat("long_identifier_", 3000) + "\n-- END"
	d.CodeTable("SQL <literal>", code)
	d.HeadingOnNewPage("Next section", 1)
	var out bytes.Buffer
	if err := d.Write(t.Context(), &out); err != nil {
		t.Fatal(err)
	}
	p := parts(t, out.Bytes())
	var doc struct {
		Body struct {
			Tables []struct {
				Properties struct {
					Width struct {
						Value int `xml:"w,attr"`
					} `xml:"tblW"`
				} `xml:"tblPr"`
				Rows []struct {
					Properties struct {
						Header  *struct{} `xml:"tblHeader"`
						NoSplit *struct {
							Value string `xml:"val,attr"`
						} `xml:"cantSplit"`
						Height *struct{} `xml:"trHeight"`
					} `xml:"trPr"`
					Cells []struct {
						Properties struct {
							Fill struct {
								Value string `xml:"fill,attr"`
							} `xml:"shd"`
						} `xml:"tcPr"`
						Paragraph struct {
							Properties struct {
								Style struct {
									Value string `xml:"val,attr"`
								} `xml:"pStyle"`
							} `xml:"pPr"`
							Runs []struct {
								Properties struct {
									Color struct {
										Value string `xml:"val,attr"`
									} `xml:"color"`
								} `xml:"rPr"`
								XML string `xml:",innerxml"`
							} `xml:"r"`
						} `xml:"p"`
					} `xml:"tc"`
				} `xml:"tr"`
			} `xml:"tbl"`
		} `xml:"body"`
	}
	if err := xml.Unmarshal([]byte(p["word/document.xml"]), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Body.Tables) != 1 || len(doc.Body.Tables[0].Rows) != 2 {
		t.Fatal("expected one header/body table")
	}
	table := doc.Body.Tables[0]
	if table.Properties.Width.Value != 9746 {
		t.Fatal("code table does not use available page width", table.Properties.Width.Value)
	}
	for i, row := range table.Rows {
		if len(row.Cells) != 1 || row.Properties.Height != nil {
			t.Fatal("code cell shape or fixed-height clipping", i)
		}
		cell := row.Cells[0]
		wantStyle := "CodeText"
		if i == 0 {
			wantStyle = "CodeHeader"
		}
		if cell.Paragraph.Properties.Style.Value != wantStyle {
			t.Fatal("code cell selected an unrelated style", i, cell.Paragraph.Properties.Style.Value)
		}
		if cell.Properties.Fill.Value != "000000" || len(cell.Paragraph.Runs) != 1 || cell.Paragraph.Runs[0].Properties.Color.Value != "FFFFFF" {
			t.Fatal("code palette lost", i)
		}
		if i == 0 {
			if row.Properties.Header == nil || row.Properties.NoSplit == nil {
				t.Fatal("header is not repeated and kept together")
			}
		} else if row.Properties.Header != nil || row.Properties.NoSplit != nil && row.Properties.NoSplit.Value != "0" {
			t.Fatal("body cannot split across pages")
		}
		want := "SQL <literal>"
		if i == 1 {
			want = strings.ReplaceAll(code, "\r\n", "\n")
		}
		if got := codeRunText(t, cell.Paragraph.Runs[0].XML); got != want {
			t.Fatalf("literal content changed: row %d got %d bytes, want %d", i, len(got), len(want))
		}
	}
	var styles struct {
		Styles []struct {
			ID        string `xml:"styleId,attr"`
			Paragraph struct {
				KeepNext  *struct{} `xml:"keepNext"`
				KeepLines *struct{} `xml:"keepLines"`
				Outline   *struct{} `xml:"outlineLvl"`
				Alignment struct {
					Value string `xml:"val,attr"`
				} `xml:"jc"`
			} `xml:"pPr"`
			Run struct {
				Font struct {
					ASCII string `xml:"ascii,attr"`
					HANSI string `xml:"hAnsi,attr"`
				} `xml:"rFonts"`
				Size struct {
					Value int `xml:"val,attr"`
				} `xml:"sz"`
				Bold  *struct{} `xml:"b"`
				Color struct {
					Value string `xml:"val,attr"`
				} `xml:"color"`
			} `xml:"rPr"`
		} `xml:"style"`
	}
	if err := xml.Unmarshal([]byte(p["word/styles.xml"]), &styles); err != nil {
		t.Fatal(err)
	}
	matched := 0
	for _, style := range styles.Styles {
		if style.ID != "CodeText" && style.ID != "CodeHeader" {
			continue
		}
		matched++
		if style.Run.Font.ASCII != "Consolas" || style.Run.Font.HANSI != "Consolas" || style.Run.Size.Value != 16 || style.Run.Color.Value != "FFFFFF" || style.Paragraph.Alignment.Value != "left" || style.Paragraph.KeepLines != nil || style.Paragraph.Outline != nil {
			t.Fatal("code style lost its typography/pagination contract", style.ID)
		}
		if (style.Run.Bold != nil) != (style.ID == "CodeHeader") || (style.Paragraph.KeepNext != nil) != (style.ID == "CodeHeader") {
			t.Fatal("header emphasis or body pagination changed", style.ID)
		}
	}
	if matched != 2 {
		t.Fatal("missing or duplicate code styles", matched)
	}
	if strings.Contains(p["word/document.xml"], `w:pStyle w:val="TableAfter"`) {
		t.Fatal("spacer before new-page heading can spill onto blank page")
	}
}

func codeRunText(t *testing.T, source string) string {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader("<root>" + source + "</root>"))
	var result strings.Builder
	text := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return result.String()
		}
		if err != nil {
			t.Fatal(err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "t":
				text = true
			case "br":
				result.WriteByte('\n')
			case "tab":
				result.WriteByte('\t')
			}
		case xml.EndElement:
			if value.Name.Local == "t" {
				text = false
			}
		case xml.CharData:
			if text {
				result.Write(value)
			}
		}
	}
}

func TestCodeTableRejectsInvalidContentWithoutWriting(t *testing.T) {
	for _, tc := range []struct{ name, header, code string }{
		{"header control", "bad\x00", "SELECT 1"},
		{"body control", "SQL", "bad\x01"},
		{"invalid UTF8", "SQL", "\xff"},
		{"text limit", "SQL", strings.Repeat("x", 64<<20)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := fixture(t)
			d.CodeTable(tc.header, tc.code)
			var out bytes.Buffer
			if err := d.Write(t.Context(), &out); err == nil || out.Len() != 0 {
				t.Fatal("invalid code emitted bytes", err, out.Len())
			}
		})
	}
	d := fixture(t)
	d.CodeTable("SQL", "SELECT 1")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var out bytes.Buffer
	if err := d.Write(ctx, &out); !errors.Is(err, context.Canceled) || out.Len() != 0 {
		t.Fatal("cancelled code emitted bytes", err)
	}
}

func TestCodeTableDoesNotChangeExistingTableStyles(t *testing.T) {
	before := fixture(t)
	before.AddTable(document.Table{Headers: []string{"H"}, Widths: []int{4000}, Rows: [][]document.Cell{{{Text: "V"}}}})
	var original bytes.Buffer
	if err := before.Write(t.Context(), &original); err != nil {
		t.Fatal(err)
	}
	before.CodeTable("", "")
	var extended bytes.Buffer
	if err := before.Write(t.Context(), &extended); err != nil {
		t.Fatal(err)
	}
	a, b := parts(t, original.Bytes()), parts(t, extended.Bytes())
	firstTable := func(s string) string {
		return s[strings.Index(s, "<w:tbl>") : strings.Index(s, "</w:tbl>")+len("</w:tbl>")]
	}
	if firstTable(a["word/document.xml"]) != firstTable(b["word/document.xml"]) {
		t.Fatal("adding a code table changed an ordinary table")
	}
	if !strings.HasPrefix(b["word/styles.xml"], strings.TrimSuffix(a["word/styles.xml"], "</w:styles>")) {
		t.Fatal("adding code definitions changed existing style definitions")
	}
}
