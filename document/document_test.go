package document_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/report"
)

func fixture(t *testing.T) *document.Document {
	t.Helper()
	d, err := document.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func parts(t *testing.T, data []byte) map[string]string {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]string{}
	for _, file := range z.File {
		r, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		dec := xml.NewDecoder(bytes.NewReader(b))
		for {
			_, err := dec.Token()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatalf("%s: %v", file.Name, err)
			}
		}
		result[file.Name] = string(b)
	}
	return result
}

func TestLiteralTextLayoutAndOwnedInputs(t *testing.T) {
	d := fixture(t)
	d.Title("Technical Review")
	d.Subtitle("Synthetic example")
	d.PageBreak()
	d.Heading("Findings", 1)
	d.Paragraph("ação 東京 😀 <w:fldSimple> & =1+1\nsoft\ttab")
	table := document.Table{Headers: []string{"Property", "Source", "Target"}, Widths: []int{3500, 3000, 3246}, Rows: [][]document.Cell{{{Text: "rows"}, {Text: "9007199254740993"}, {Text: "9007199254740994", Tone: document.Difference}}}}
	d.AddTable(table)
	table.Rows[0][1].Text = "MUTATED"
	table.Widths[0] = 0
	var out bytes.Buffer
	if err := d.Write(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	p := parts(t, out.Bytes())
	main := p["word/document.xml"]
	for _, text := range []string{"9007199254740993", "&lt;w:fldSimple&gt;", "&amp;", "ação 東京 😀", "w:tblHeader", "w:gridCol w:w=\"3500\"", "w:type=\"fixed\"", "w:br/>", "w:tab/>", "FFF0D5"} {
		if !strings.Contains(main, text) {
			t.Fatalf("missing %q", text)
		}
	}
	if strings.Contains(main, "MUTATED") || strings.Contains(main, "<w:fldSimple>") {
		t.Fatal("caller mutation or literal text changed document structure")
	}
	if !strings.Contains(p["word/footer1.xml"], `w:instr="PAGE"`) || !strings.Contains(p["word/settings.xml"], "doNotExpandShiftReturn") {
		t.Fatal("page field or soft-break policy missing")
	}
	if !strings.Contains(p["word/styles.xml"], `w:sz w:val="19"`) {
		t.Fatal("report body default changed")
	}
}

func TestInvalidInputWritesNothing(t *testing.T) {
	for name, configure := range map[string]func(*document.Document){
		"control":          func(d *document.Document) { d.Paragraph("bad\x01") },
		"utf8":             func(d *document.Document) { d.Paragraph(string([]byte{0xff})) },
		"heading":          func(d *document.Document) { d.Heading("bad", 4) },
		"heading-new-page": func(d *document.Document) { d.HeadingOnNewPage("bad", 4) },
		"width":            func(d *document.Document) { d.AddTable(document.Table{Headers: []string{"a"}, Widths: []int{30000}}) },
		"shape": func(d *document.Document) {
			d.AddTable(document.Table{Headers: []string{"a"}, Widths: []int{1000}, Rows: [][]document.Cell{{}}})
		},
		"tone": func(d *document.Document) {
			d.AddTable(document.Table{Headers: []string{"a"}, Widths: []int{1000}, Rows: [][]document.Cell{{{Tone: "unknown"}}}})
		},
	} {
		t.Run(name, func(t *testing.T) {
			d := fixture(t)
			configure(d)
			var out bytes.Buffer
			if err := d.Write(context.Background(), &out); err == nil || out.Len() != 0 {
				t.Fatal("invalid input emitted bytes")
			}
		})
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

type cancellingWriter struct{ cancel context.CancelFunc }

func (w cancellingWriter) Write(p []byte) (int, error) { w.cancel(); return len(p), nil }

func TestCancellationAndWriterErrors(t *testing.T) {
	d := fixture(t)
	d.Paragraph("sample")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	if err := d.Write(ctx, &out); !errors.Is(err, context.Canceled) || out.Len() != 0 {
		t.Fatal("pre-cancelled write emitted data")
	}
	want := errors.New("writer failed")
	if err := d.Write(context.Background(), failingWriter{want}); !errors.Is(err, want) {
		t.Fatalf("writer cause lost: %v", err)
	}
	if err := d.Write(context.Background(), shortWriter{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write hidden: %v", err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	if err := d.Write(ctx, cancellingWriter{cancel}); !errors.Is(err, context.Canceled) {
		t.Fatalf("in-progress cancellation hidden: %v", err)
	}
}

func TestOptionsRejectInvalidGeometry(t *testing.T) {
	o := report.Defaults()
	o.Margin = o.PageWidth
	if _, err := document.New(o); err == nil {
		t.Fatal("invalid margins admitted")
	}
	o = report.Defaults()
	o.TableSize = 0
	if _, err := document.New(o); err == nil {
		t.Fatal("invalid size admitted")
	}
}

func TestHeadingOnNewPageKeepsBoundaryAfterFullTable(t *testing.T) {
	d := fixture(t)
	rows := make([][]document.Cell, 100)
	for i := range rows {
		rows[i] = []document.Cell{{Text: "Long preceding inventory"}, {Text: "Exact row content"}}
	}
	d.AddTable(document.Table{Headers: []string{"Name", "Value"}, Widths: []int{4000, 5000}, Rows: rows})
	d.HeadingOnNewPage("Checks <literal>", 2)
	d.Heading("First check", 3)
	var out bytes.Buffer
	if err := d.Write(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	xmlParts := parts(t, out.Bytes())
	var tree struct {
		Body struct {
			Paragraphs []struct {
				Properties struct {
					Style struct {
						Value string `xml:"val,attr"`
					} `xml:"pStyle"`
					NewPage *struct{} `xml:"pageBreakBefore"`
				} `xml:"pPr"`
				Runs []struct {
					Text   string `xml:"t"`
					Breaks []struct {
						Type string `xml:"type,attr"`
					} `xml:"br"`
				} `xml:"r"`
			} `xml:"p"`
		} `xml:"body"`
	}
	if err := xml.Unmarshal([]byte(xmlParts["word/document.xml"]), &tree); err != nil {
		t.Fatal(err)
	}
	ps := tree.Body.Paragraphs
	if len(ps) != 2 {
		t.Fatal("full table acquired a spacer or break-only paragraph", len(ps))
	}
	if ps[0].Properties.Style.Value != "Heading2" || ps[0].Properties.NewPage == nil || len(ps[0].Runs) != 1 || ps[0].Runs[0].Text != "Checks <literal>" {
		t.Fatal("new-page boundary or literal heading text was lost")
	}
	if ps[1].Properties.Style.Value != "Heading3" || ps[1].Properties.NewPage != nil {
		t.Fatal("child heading inherited a new page")
	}
	for _, p := range ps {
		for _, r := range p.Runs {
			for _, br := range r.Breaks {
				if br.Type == "page" {
					t.Fatal("standalone break can spill after a full table")
				}
			}
		}
	}
}
