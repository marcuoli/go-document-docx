package editing_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"archive/zip"
	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/report"
	"github.com/marcuoli/go-document-docx/tabledata"
	"io"
)

func parts(t *testing.T, data []byte) map[string]string {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		_ = r.Close()
		out[f.Name] = string(b)
	}
	return out
}
func TestOwnedAuthoringAndWordControls(t *testing.T) {
	d, err := editing.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	heading, err := d.Heading("Technical review", 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := heading.Bookmark("overview"); err != nil {
		t.Fatal(err)
	}
	p, err := d.Paragraph("Exact <&> 9007199254740993\nSecond\tcolumn", editing.ParagraphOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Link("Overview", "overview", "Go to overview", true); err != nil {
		t.Fatal(err)
	}
	if err := p.Link("Reference", "https://example.com/reference", "Read reference", false); err != nil {
		t.Fatal(err)
	}
	if _, err := d.List("First action", true, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Contents(3); err != nil {
		t.Fatal(err)
	}
	data := tabledata.Data{Headers: []string{"Name", "Value"}, Rows: [][]string{{"A_LONG_IDENTIFIER_WITH_SUFFIX", "9007199254740993"}}}
	_, err = d.Table(data, editing.TableOptions{Columns: []editing.Column{{Width: "35%", SingleLine: true, Ellipsis: 12}, {Width: "65%"}}})
	if err != nil {
		t.Fatal(err)
	}
	data.Rows[0][1] = "mutated"
	var output bytes.Buffer
	if err := d.Write(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	xml := parts(t, output.Bytes())
	for _, expected := range []string{"9007199254740993", "w:anchor=\"overview\"", "w:tooltip=\"A_LONG_IDENTIFIER_WITH_SUFFIX\"", "w:type=\"fixed\"", "w:tblHeader", "w:cantSplit", "w:br", "w:tab", "w:numPr", "TOC"} {
		if !strings.Contains(xml["word/document.xml"], expected) {
			t.Fatalf("missing %s", expected)
		}
	}
	if strings.Contains(xml["word/document.xml"], "mutated") {
		t.Fatal("caller input alias")
	}
	if !strings.Contains(xml["word/settings.xml"], "doNotExpandShiftReturn") {
		t.Fatal("spacing compatibility lost")
	}
	if !strings.Contains(xml["word/numbering.xml"], "abstractNum") {
		t.Fatal("numbering missing")
	}
	if !strings.Contains(xml["word/_rels/document.xml.rels"], "https://example.com/reference") {
		t.Fatal("link relationship missing")
	}
}
func TestWriteRejectsDuplicateBookmarkAndCancellation(t *testing.T) {
	d, err := editing.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		p, err := d.Paragraph("Text", editing.ParagraphOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Bookmark("same"); err != nil {
			t.Fatal(err)
		}
	}
	var b bytes.Buffer
	if err := d.Write(context.Background(), &b); err == nil || b.Len() != 0 {
		t.Fatal("duplicate written", err, b.Len())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := d.Write(ctx, &b); err != context.Canceled || b.Len() != 0 {
		t.Fatal(err, b.Len())
	}
}
func TestCaptionCountersAndExplicitZero(t *testing.T) {
	for range 2 {
		d, err := editing.New(report.Defaults())
		if err != nil {
			t.Fatal(err)
		}
		p, err := d.Caption("Table", "Metrics")
		if err != nil {
			t.Fatal(err)
		}
		if p.Native().Text() != "Table 1: Metrics" {
			t.Fatal(p.Native().Text())
		}
	}
	d, _ := editing.New(report.Defaults())
	p, err := d.Code("SELECT * FROM dual")
	if err != nil {
		t.Fatal(err)
	}
	if p.Native().Alignment() != 0 {
		t.Fatal("left alignment ignored")
	}
	if _, err := d.Paragraph("bad\x01", editing.ParagraphOptions{}); err == nil {
		t.Fatal("invalid character admitted")
	}
}
