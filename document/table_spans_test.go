package document_test

import (
	"bytes"
	"encoding/xml"
	"math"
	"slices"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
)

func TestGroupedTableGridAndHeaderOwnership(t *testing.T) {
	d := fixture(t)
	table := document.Table{
		Widths: []int{1200, 900, 1100, 700, 1300, 800},
		HeaderRows: [][]document.Cell{
			{{}, {Runs: []document.CellRun{{Text: "Database A"}}, ColSpan: 2, Alignment: "center", Bookmark: "database_a"}, {Text: "Database B", ColSpan: 2}, {}},
			{{Text: "Property"}, {Text: "Zone"}, {Text: "Time"}, {Text: "Zone"}, {Text: "Time"}, {Text: "Result"}},
		},
		Rows: [][]document.Cell{
			{{Text: "Time zone"}, {Text: "+00:00", ColSpan: 2, LinkTarget: "database_a"}, {Text: "+00:00", ColSpan: 2}, {Text: "✓"}},
		},
	}
	d.AddTable(table)
	table.Widths[2] = 0
	table.HeaderRows[0][1].Runs[0].Text = "changed"
	table.HeaderRows[0][1].Bookmark = "changed"
	table.HeaderRows[1][1].Text = "changed"
	table.Rows[0][1].ColSpan = 0
	var b bytes.Buffer
	if err := d.Write(t.Context(), &b); err != nil {
		t.Fatal(err)
	}
	type cell struct {
		Width struct {
			Twips int `xml:"w,attr"`
		} `xml:"tcPr>tcW"`
		Span struct {
			Count int `xml:"val,attr"`
		} `xml:"tcPr>gridSpan"`
	}
	var parsed struct {
		Rows []struct {
			Header *struct{} `xml:"trPr>tblHeader"`
			Cells  []cell    `xml:"tc"`
		} `xml:"body>tbl>tr"`
	}
	if err := xml.Unmarshal([]byte(parts(t, b.Bytes())["word/document.xml"]), &parsed); err != nil {
		t.Fatal(err)
	}
	wantWidths := [][]int{{1200, 2000, 2000, 800}, {1200, 900, 1100, 700, 1300, 800}, {1200, 2000, 2000, 800}}
	wantSpans := [][]int{{0, 2, 2, 0}, {0, 0, 0, 0, 0, 0}, {0, 2, 2, 0}}
	if len(parsed.Rows) != len(wantWidths) {
		t.Fatal("unexpected row count", len(parsed.Rows))
	}
	for i, row := range parsed.Rows {
		widths, spans := make([]int, 0, len(row.Cells)), make([]int, 0, len(row.Cells))
		for _, cell := range row.Cells {
			widths = append(widths, cell.Width.Twips)
			spans = append(spans, cell.Span.Count)
		}
		if (row.Header != nil) != (i < 2) || !slices.Equal(widths, wantWidths[i]) || !slices.Equal(spans, wantSpans[i]) {
			t.Fatalf("row %d: header %t, widths %v, spans %v", i, row.Header != nil, widths, spans)
		}
	}
	nav := inspectNavigation(t, b.Bytes())
	if nav.Text["database_a"] != "Database A" || len(nav.Links) != 1 || nav.Links[0].Target != "database_a" {
		t.Fatalf("header ownership or navigation changed: %+v", nav)
	}
}

func TestMalformedTableSpansWriteNothing(t *testing.T) {
	for _, tc := range []struct {
		name             string
		headers          []string
		headerRows, rows [][]document.Cell
	}{
		{name: "both header forms", headers: []string{"A", "B"}, headerRows: [][]document.Cell{{{ColSpan: 2}}}},
		{name: "no header"},
		{name: "empty header row", headerRows: [][]document.Cell{{}}},
		{name: "short header", headerRows: [][]document.Cell{{{Text: "A"}}}},
		{name: "span overflow", headers: []string{"A", "B"}, rows: [][]document.Cell{{{ColSpan: math.MaxInt}}}},
		{name: "negative span", headers: []string{"A", "B"}, rows: [][]document.Cell{{{ColSpan: -1}, {}}}},
		{name: "extra covered cell", headers: []string{"A", "B"}, rows: [][]document.Cell{{{ColSpan: 2}, {}}}},
		{name: "short body", headers: []string{"A", "B"}, rows: [][]document.Cell{{{ColSpan: 1}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := fixture(t)
			d.AddTable(document.Table{Headers: tc.headers, Widths: []int{1000, 1000}, HeaderRows: tc.headerRows, Rows: tc.rows})
			var b bytes.Buffer
			if err := d.Write(t.Context(), &b); err == nil || b.Len() != 0 {
				t.Fatalf("malformed layout returned %v after %d bytes", err, b.Len())
			}
		})
	}
}
