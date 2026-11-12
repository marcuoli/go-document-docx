package document_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/report"
)

func TestTableAppearanceOwnsSettingsAndPreservesSemanticCells(t *testing.T) {
	options := report.Defaults()
	d, err := document.New(options)
	if err != nil {
		t.Fatal(err)
	}
	options.TableStyle.HeaderFill = "FF00FF"
	d.AddTable(document.Table{Headers: []string{"Field", "Value"}, Widths: []int{4000, 5746}, Rows: [][]document.Cell{
		{{Text: "unchanged & literal"}, {Text: "✓", TextColor: "2E7D32"}},
		{{Text: "missing"}, {Text: "N/A", TextColor: "B3261E"}},
		{{Text: "different"}, {Text: "9007199254740993", Tone: document.Difference}},
	}})
	var output bytes.Buffer
	if err := d.Write(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	xml := parts(t, output.Bytes())
	body := xml["word/document.xml"]
	for _, want := range []string{
		`w:fill="000000"`, `w:color w:val="FFFFFF"`, `w:vAlign w:val="center"`,
		`<w:top w:w="0" w:type="dxa"/>`, `<w:left w:w="50" w:type="dxa"/>`,
		`<w:insideV w:val="single" w:sz="4" w:color="000000"/>`,
		`w:val="2E7D32"`, `w:val="B3261E"`, `w:fill="FFF0D5"`,
		"9007199254740993", "unchanged &amp; literal",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing appearance/data property %q", want)
		}
	}
	if strings.Contains(body, "FF00FF") || strings.Contains(body, "tcFitText") || strings.Contains(body, "trHeight") {
		t.Fatal("aliased settings or forced text/row fitting")
	}
	if !strings.Contains(xml["word/styles.xml"], `<w:spacing w:before="0" w:after="0" w:line="240" w:lineRule="auto"/>`) {
		t.Fatal("table spacing was not explicit")
	}
}

func TestNilTableAppearanceKeepsLegacyRendering(t *testing.T) {
	options := report.Defaults()
	options.TableStyle = nil
	d, err := document.New(options)
	if err != nil {
		t.Fatal(err)
	}
	d.AddTable(document.Table{Headers: []string{"Field"}, Widths: []int{9746}, Rows: [][]document.Cell{{{Text: "row"}}, {{Text: "second"}}}})
	var output bytes.Buffer
	if err := d.Write(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	body := parts(t, output.Bytes())["word/document.xml"]
	for _, want := range []string{`w:fill="E9E5DF"`, `w:fill="F7F6F4"`, `w:color="E8E5E1"`, `<w:top w:w="70"`, `w:vAlign w:val="top"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("legacy appearance changed: %q", want)
		}
	}
}

func TestTableAppearanceRejectsInvalidOptions(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*document.TableStyle)
	}{
		{"bad-rgb", func(s *document.TableStyle) { s.HeaderFill = "#000000" }},
		{"invalid-hex", func(s *document.TableStyle) { s.BorderColor = "GGGGGG" }},
		{"negative-padding", func(s *document.TableStyle) { s.PaddingLeft = -1 }},
		{"large-spacing", func(s *document.TableStyle) { s.ParagraphAfter = 31681 }},
		{"alignment", func(s *document.TableStyle) { s.VerticalAlignment = "justify" }},
		{"font-size-limit", func(s *document.TableStyle) { s.HeaderFont = strings.Repeat("a", 256) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := report.Defaults()
			tc.edit(o.TableStyle)
			if _, err := document.New(o); err == nil {
				t.Fatal("invalid appearance admitted")
			}
		})
	}
}
