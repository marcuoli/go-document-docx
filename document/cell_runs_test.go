package document_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
)

func TestCellRunsPreserveMixedLiteralLinesAndOwnership(t *testing.T) {
	d := fixture(t)
	runs := []document.CellRun{{Text: "1:N/A\r\n"}, {Text: "2:N/A", TextColor: "B3261E"}, {Text: "\n\t ação 東京 😀 <&>"}}
	d.AddTable(document.Table{Headers: []string{"Value"}, Widths: []int{5000}, Rows: [][]document.Cell{{{Runs: runs, TextColor: "007F00", Bold: true, Tone: document.Difference}}}})
	runs[0].Text = "MUTATED"
	runs[1].TextColor = "not a color"
	var out bytes.Buffer
	if err := d.Write(t.Context(), &out); err != nil {
		t.Fatal(err)
	}
	p := parts(t, out.Bytes())
	var doc struct {
		Body struct {
			Tables []struct {
				Rows []struct {
					Cells []struct {
						Properties struct {
							Shading struct {
								Fill string `xml:"fill,attr"`
							} `xml:"shd"`
						} `xml:"tcPr"`
						Paragraphs []struct {
							Runs []struct {
								Properties struct {
									Bold  *struct{} `xml:"b"`
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
	cell := doc.Body.Tables[0].Rows[1].Cells[0]
	if cell.Properties.Shading.Fill != "FFF0D5" || len(cell.Paragraphs) != 1 || len(cell.Paragraphs[0].Runs) != 3 {
		t.Fatal("runs changed cell geometry, fill or paragraph structure")
	}
	for i, run := range cell.Paragraphs[0].Runs {
		wantText := []string{"1:N/A\n", "2:N/A", "\n\t ação 東京 😀 <&>"}[i]
		wantColor := []string{"007F00", "B3261E", "007F00"}[i]
		if got := codeRunText(t, run.XML); got != wantText || run.Properties.Color.Value != wantColor || run.Properties.Bold == nil {
			t.Fatal("literal run, inherited color or all-run bold changed", i, got, run.Properties.Color.Value)
		}
	}
}

func TestCellRunsRetainLegacySerialization(t *testing.T) {
	for _, tc := range []struct {
		name         string
		legacy, rich document.Cell
	}{
		{"inherited", document.Cell{Text: "Exact\n\t<&>"}, document.Cell{Runs: []document.CellRun{{Text: "Exact\n\t<&>"}}}},
		{"cell formatting", document.Cell{Text: "1:N/A", TextColor: "007F00", Bold: true}, document.Cell{Runs: []document.CellRun{{Text: "1:N/A"}}, TextColor: "007F00", Bold: true}},
		{"explicit run color", document.Cell{Text: "2:N/A", TextColor: "B3261E"}, document.Cell{Runs: []document.CellRun{{Text: "2:N/A", TextColor: "B3261E"}}}},
		{"empty runs", document.Cell{Text: "legacy"}, document.Cell{Text: "legacy", Runs: []document.CellRun{}}},
		{"empty text", document.Cell{}, document.Cell{Runs: []document.CellRun{{}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serialize := func(cell document.Cell) []byte {
				d := fixture(t)
				d.AddTable(document.Table{Headers: []string{"Value"}, Widths: []int{5000}, Rows: [][]document.Cell{{cell}}})
				var out bytes.Buffer
				if err := d.Write(t.Context(), &out); err != nil {
					t.Fatal(err)
				}
				return out.Bytes()
			}
			if !bytes.Equal(serialize(tc.legacy), serialize(tc.rich)) {
				t.Fatal("equivalent legacy/rich cells serialized differently")
			}
		})
	}
}

func TestCellRunsValidateBeforeOutput(t *testing.T) {
	for _, tc := range []struct {
		name string
		cell document.Cell
	}{
		{"ambiguous text", document.Cell{Text: "keep me", Runs: []document.CellRun{{Text: "also keep me"}}}},
		{"whitespace text", document.Cell{Text: " ", Runs: []document.CellRun{{Text: "value"}}}},
		{"short color", document.Cell{Runs: []document.CellRun{{TextColor: "FFF"}}}},
		{"color injection", document.Cell{Runs: []document.CellRun{{TextColor: "#FF0000"}}}},
		{"nonhex color", document.Cell{Runs: []document.CellRun{{TextColor: "GG0000"}}}},
		{"invalid inherited color", document.Cell{TextColor: "bad", Runs: []document.CellRun{{TextColor: "FFFFFF"}}}},
		{"invalid UTF8", document.Cell{Runs: []document.CellRun{{Text: "\xff"}}}},
		{"invalid XML", document.Cell{Runs: []document.CellRun{{Text: "bad\x00value"}}}},
		{"aggregate limit", document.Cell{Runs: []document.CellRun{{Text: strings.Repeat("x", 32<<20)}, {Text: strings.Repeat("x", 32<<20)}}}},
		{"expanded XML limit", document.Cell{Runs: []document.CellRun{{Text: strings.Repeat("\t", 5<<20)}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := fixture(t)
			d.AddTable(document.Table{Headers: []string{"Value"}, Widths: []int{5000}, Rows: [][]document.Cell{{tc.cell}}})
			var out bytes.Buffer
			if err := d.Write(t.Context(), &out); err == nil || out.Len() != 0 {
				t.Fatal("invalid runs sent bytes to writer", err, out.Len())
			}
		})
	}
}

func TestCellRunsNormalizeCrossRunCRLF(t *testing.T) {
	for _, tc := range []struct {
		name  string
		texts []string
	}{
		{"split pair", []string{"a\r", "\nb"}},
		{"empty span", []string{"a\r", "", "\nb"}},
		{"LF remains separate", []string{"a\n", "\nb"}},
		{"consecutive pairs", []string{"\r", "\n\r", "\n"}},
		{"trailing CR", []string{"a\r", ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runs := make([]document.CellRun, len(tc.texts))
			for i, text := range tc.texts {
				runs[i] = document.CellRun{Text: text, TextColor: "B3261E"}
			}
			d := fixture(t)
			d.AddTable(document.Table{Headers: []string{"Value"}, Widths: []int{5000}, Rows: [][]document.Cell{{{Runs: runs}}}})
			var out bytes.Buffer
			if err := d.Write(t.Context(), &out); err != nil {
				t.Fatal(err)
			}
			want := strings.ReplaceAll(strings.ReplaceAll(strings.Join(tc.texts, ""), "\r\n", "\n"), "\r", "\n")
			if got := codeRunText(t, parts(t, out.Bytes())["word/document.xml"]); got != "Value"+want {
				t.Fatalf("run boundary changed literal line breaks: %q, want %q", got, "Value"+want)
			}
		})
	}
}

func TestCellRunsPreserveCancellationAndWriterErrors(t *testing.T) {
	d := fixture(t)
	d.AddTable(document.Table{Headers: []string{"Value"}, Widths: []int{5000}, Rows: [][]document.Cell{{{Runs: []document.CellRun{{Text: "1:N/A\n"}, {Text: "2:N/A", TextColor: "B3261E"}}}}}})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var out bytes.Buffer
	if err := d.Write(ctx, &out); !errors.Is(err, context.Canceled) || out.Len() != 0 {
		t.Fatal("cancelled runs produced output", err)
	}
	sentinel := errors.New("writer failed")
	if err := d.Write(t.Context(), cellRunErrorWriter{sentinel}); !errors.Is(err, sentinel) {
		t.Fatal("writer error identity lost", err)
	}
}

type cellRunErrorWriter struct{ err error }

func (w cellRunErrorWriter) Write([]byte) (int, error) { return 0, w.err }
