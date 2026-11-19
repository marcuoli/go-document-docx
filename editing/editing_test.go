package editing_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"archive/zip"
	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/internal/opc"
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

func TestIndependentListsPreserveLegacySharedSequence(t *testing.T) {
	d, err := editing.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.List("Shared first", true, 0); err != nil {
		t.Fatal(err)
	}
	independent := d.NewList(true)
	if _, err := independent.Item("Independent first", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := d.List("Shared second", true, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := independent.Item("Independent second", 0); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := d.Write(t.Context(), &b); err != nil {
		t.Fatal(err)
	}
	pkg, err := opc.Read(t.Context(), b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	xml, err := pkg.XML("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, p := range opc.Elements(xml.Root(), opc.Word, "p") {
		pr := opc.Child(p, opc.Word, "pPr")
		if pr == nil {
			continue
		}
		num := opc.Child(pr, opc.Word, "numPr")
		if num != nil {
			ids = append(ids, opc.Attr(opc.Child(num, opc.Word, "numId"), opc.Word, "val"))
		}
	}
	if len(ids) != 4 || ids[0] != ids[2] || ids[1] != ids[3] || ids[0] == ids[1] {
		t.Fatal("shared/independent sequence ownership changed", ids)
	}
	var uninitialized editing.List
	if _, err := uninitialized.Item("Invalid", 0); err == nil {
		t.Fatal("zero list accepted")
	}
	if _, err := independent.Item("Invalid", 9); err == nil {
		t.Fatal("invalid list level accepted")
	}
}

func TestWriteRejectsCaseVariantBookmarks(t *testing.T) {
	d, err := editing.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"CaseTarget", "casetarget"} {
		p, err := d.Paragraph(name, editing.ParagraphOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Bookmark(name); err != nil {
			t.Fatal(err)
		}
	}
	var b bytes.Buffer
	if err := d.Write(t.Context(), &b); err == nil || b.Len() != 0 {
		t.Fatal("case-variant duplicates published", err, b.Len())
	}
}
func TestGeneratedCellBookmarkAvoidsCaseVariantCallerName(t *testing.T) {
	d, err := editing.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Paragraph("Reserved", editing.ParagraphOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Bookmark("CELL_VALUE_2"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Table(tabledata.Data{Headers: []string{"Name"}, Rows: [][]string{{"A_LONG_LITERAL_IDENTIFIER"}}}, editing.TableOptions{Columns: []editing.Column{{Width: "100%", SingleLine: true, Ellipsis: 8}}}); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := d.Write(t.Context(), &b); err != nil {
		t.Fatal(err)
	}
	pkg, err := opc.Read(t.Context(), b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	doc, err := pkg.XML("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, mark := range opc.Elements(doc.Root(), opc.Word, "bookmarkStart") {
		key := strings.ToLower(opc.Attr(mark, opc.Word, "name"))
		if seen[key] {
			t.Fatal("case-variant tooltip collision", key)
		}
		seen[key] = true
	}
	if !seen["cell_value_2"] || !seen["cell_value_3"] {
		t.Fatal("caller or generated bookmark missing", seen)
	}
}

func TestLoadedBookmarkRejectsCaseVariantAddition(t *testing.T) {
	d, err := editing.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Paragraph("Existing", editing.ParagraphOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Bookmark("CaseTarget"); err != nil {
		t.Fatal(err)
	}
	var original bytes.Buffer
	if err := d.Write(t.Context(), &original); err != nil {
		t.Fatal(err)
	}
	loaded, err := editing.Open(t.Context(), original.Bytes(), report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	added, err := loaded.Paragraph("Added", editing.ParagraphOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := added.Bookmark("casetarget"); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := loaded.Write(t.Context(), &b); err == nil || !strings.Contains(err.Error(), "duplicate bookmark") || b.Len() != 0 {
		t.Fatal("loaded case collision not rejected", err, b.Len())
	}
}

func TestAuthoredLinkSpellingIsExactOnBothEditorPaths(t *testing.T) {
	for _, route := range []string{"paragraph", "table"} {
		for _, wrongCase := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/wrong_case_%t", route, wrongCase), func(t *testing.T) {
				d, err := editing.New(report.Defaults())
				if err != nil {
					t.Fatal(err)
				}
				destination, err := d.Paragraph("Destination", editing.ParagraphOptions{})
				if err != nil {
					t.Fatal(err)
				}
				if err := destination.Bookmark("MixedCase"); err != nil {
					t.Fatal(err)
				}
				target := "MixedCase"
				if wrongCase {
					target = "mixedcase"
				}
				if route == "paragraph" {
					p, err := d.Paragraph("", editing.ParagraphOptions{})
					if err != nil {
						t.Fatal(err)
					}
					if err := p.Link("Go", target, "", true); err != nil {
						t.Fatal(err)
					}
				} else {
					table, err := d.Table(tabledata.Data{Headers: []string{"Value"}, Rows: [][]string{{"Go"}}}, editing.TableOptions{Columns: []editing.Column{{Width: "100%"}}})
					if err != nil {
						t.Fatal(err)
					}
					if err := table.Link(1, 0, "Go", target, "", true); err != nil {
						t.Fatal(err)
					}
				}
				var b bytes.Buffer
				err = d.Write(t.Context(), &b)
				if wrongCase {
					if err == nil || b.Len() != 0 {
						t.Fatal("wrong-case target published", err, b.Len())
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				pkg, err := opc.Read(t.Context(), b.Bytes())
				if err != nil {
					t.Fatal(err)
				}
				doc, err := pkg.XML("word/document.xml")
				if err != nil {
					t.Fatal(err)
				}
				bookmark, anchor := false, false
				for _, mark := range opc.Elements(doc.Root(), opc.Word, "bookmarkStart") {
					if opc.Attr(mark, opc.Word, "name") == "MixedCase" {
						bookmark = true
					}
				}
				for _, link := range opc.Elements(doc.Root(), opc.Word, "hyperlink") {
					if opc.Attr(link, opc.Word, "anchor") == "MixedCase" {
						anchor = true
					} else {
						t.Fatal("serialized anchor spelling changed", opc.Attr(link, opc.Word, "anchor"))
					}
				}
				if !bookmark || !anchor {
					t.Fatal("exact serialized bookmark/anchor missing", bookmark, anchor)
				}
			})
		}
	}
}
