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

type navigationXML struct {
	Names      map[string]string
	Text       map[string]string
	Links      []navigationLink
	Paragraphs int
}
type navigationLink struct {
	Target, Tip, Text string
	Nested            bool
	Underlined        bool
	Color             string
}

func inspectNavigation(t *testing.T, data []byte) navigationXML {
	t.Helper()
	main := parts(t, data)["word/document.xml"]
	result := navigationXML{Names: map[string]string{}, Text: map[string]string{}}
	active := map[string]string{}
	ends := map[string]bool{}
	decoder := xml.NewDecoder(strings.NewReader(main))
	link := -1
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		switch e := token.(type) {
		case xml.StartElement:
			attrs := map[string]string{}
			for _, a := range e.Attr {
				attrs[a.Name.Local] = a.Value
			}
			switch e.Name.Local {
			case "p":
				result.Paragraphs++
			case "bookmarkStart":
				if _, exists := active[attrs["id"]]; exists {
					t.Fatal("duplicate ID")
				}
				if _, exists := result.Names[attrs["name"]]; exists {
					t.Fatal("duplicate name")
				}
				active[attrs["id"]] = attrs["name"]
				result.Names[attrs["name"]] = attrs["id"]
			case "bookmarkEnd":
				if _, exists := active[attrs["id"]]; !exists || ends[attrs["id"]] {
					t.Fatal("unpaired/duplicate end")
				}
				delete(active, attrs["id"])
				ends[attrs["id"]] = true
			case "hyperlink":
				if link >= 0 {
					t.Fatal("nested hyperlink")
				}
				if attrs["id"] != "" {
					t.Fatal("internal link acquired relationship")
				}
				result.Links = append(result.Links, navigationLink{Target: attrs["anchor"], Tip: attrs["tooltip"]})
				link = len(result.Links) - 1
			case "u":
				if link >= 0 && attrs["val"] == "single" {
					result.Links[link].Underlined = true
				}
			case "color":
				if link >= 0 {
					result.Links[link].Color = attrs["val"]
				}
			case "t":
				var value string
				if err := decoder.DecodeElement(&value, &e); err != nil {
					t.Fatal(err)
				}
				for _, name := range active {
					result.Text[name] += value
				}
				if link >= 0 {
					result.Links[link].Text += value
				}
			}
		case xml.EndElement:
			if e.Name.Local == "hyperlink" {
				link = -1
			}
		}
	}
	if len(active) != 0 || len(ends) != len(result.Names) {
		t.Fatal("unpaired bookmarks", active, ends)
	}
	for _, l := range result.Links {
		if _, exists := result.Names[l.Target]; !exists {
			t.Fatal("unresolved anchor", l.Target)
		}
	}
	return result
}

func TestLiteralNavigationForwardTargetsAndOwnership(t *testing.T) {
	d := fixture(t)
	table := document.Table{Headers: []string{"Section", "Detail"}, Widths: []int{4000, 4000}, Rows: [][]document.Cell{
		{{Text: "Go <&> detail", LinkTarget: "detail", ScreenTip: "Jump <&> detail"}, {Text: "First detail", Bookmark: "cell_detail"}},
		{{Runs: []document.CellRun{{Text: "Rich\r", Italic: true}, {Text: "\nlink", TextColor: "B3261E"}}, LinkTarget: "cell_detail", Bold: true}, {Text: "Self tooltip", ScreenTip: "Present"}},
	}}
	d.AddTable(table)
	table.Rows[0][0].LinkTarget = "changed"
	table.Rows[0][1].Bookmark = "changed"
	d.BookmarkNext("detail")
	d.BookmarkNext("_same_location")
	d.Heading("Exact detail", 2)
	var out bytes.Buffer
	if err := d.Write(t.Context(), &out); err != nil {
		t.Fatal(err)
	}
	got := inspectNavigation(t, out.Bytes())
	if got.Text["detail"] != "Exact detail" || got.Text["_same_location"] != "Exact detail" || got.Text["cell_detail"] != "First detail" {
		t.Fatal("wrong target text", got.Text)
	}
	if len(got.Links) != 3 || got.Links[0].Target != "detail" || got.Links[0].Tip != "Jump <&> detail" || got.Links[0].Text != "Go <&> detail" || !got.Links[0].Underlined || got.Links[0].Color != "315357" {
		t.Fatal("navigation contract", got.Links)
	}
	if got.Links[1].Target != "cell_detail" || !got.Links[1].Underlined || got.Links[1].Color != "B3261E" {
		t.Fatal("rich link/color lost", got.Links)
	}
	if got.Links[2].Underlined {
		t.Fatal("tooltip-only gained navigation styling")
	}
	var again bytes.Buffer
	if err := d.Write(t.Context(), &again); err != nil || !bytes.Equal(out.Bytes(), again.Bytes()) {
		t.Fatal("serialization mutated document", err)
	}
}

func TestBookmarkNextAddsNoParagraphAndSkipsTablesAndBreaks(t *testing.T) {
	makeDoc := func(anchor bool) []byte {
		d := fixture(t)
		if anchor {
			d.BookmarkNext("after_table")
		}
		d.AddTable(document.Table{Headers: []string{"H"}, Widths: []int{4000}, Rows: [][]document.Cell{{{Text: "table"}}}})
		d.PageBreak()
		d.HeadingOnNewPage("Target", 1)
		var b bytes.Buffer
		if err := d.Write(t.Context(), &b); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	before, after := makeDoc(false), makeDoc(true)
	a, b := inspectNavigation(t, before), inspectNavigation(t, after)
	if a.Paragraphs != b.Paragraphs || b.Text["after_table"] != "Target" {
		t.Fatal("anchor affected layout or wrong paragraph", a, b)
	}
}

func TestLiteralNavigationGeneratedNamesCannotCollide(t *testing.T) {
	d := fixture(t)
	d.BookmarkNext("_docxcelltip2")
	d.Paragraph("Reserved caller name")
	d.AddTable(document.Table{Headers: []string{"H"}, Widths: []int{4000}, Rows: [][]document.Cell{
		{{Text: "tip", ScreenTip: "tip"}}, {{Text: "named tip", Bookmark: "with_tip", ScreenTip: "tip"}}, {{Text: "navigation+tip", Bookmark: "own", LinkTarget: "_docxcelltip2", ScreenTip: "tip"}},
	}})
	var b bytes.Buffer
	if err := d.Write(t.Context(), &b); err != nil {
		t.Fatal(err)
	}
	got := inspectNavigation(t, b.Bytes())
	folded := map[string]bool{}
	for name := range got.Names {
		key := strings.ToLower(name)
		if folded[key] {
			t.Fatal("case-variant generated bookmark collision", got.Names)
		}
		folded[key] = true
	}
	if len(got.Names) != 5 || len(got.Links) != 3 {
		t.Fatalf("unexpected bookmark or hyperlink count: %+v", got)
	}
	if got.Text["_docxcelltip2"] != "Reserved caller name" || got.Text["with_tip"] != "named tip" {
		t.Fatalf("authored bookmark moved: %+v", got.Text)
	}
	if !strings.HasPrefix(got.Links[0].Target, "_DocxCellTip") ||
		!strings.HasPrefix(got.Links[1].Target, "_DocxCellTip") ||
		got.Links[0].Target == got.Links[1].Target ||
		got.Links[2].Target != "_docxcelltip2" {
		t.Fatal(got)
	}
}

func TestLiteralNavigationRejectsBeforeOutput(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func(*document.Document)
	}{
		{"invalid paragraph", func(d *document.Document) { d.BookmarkNext("1bad"); d.Paragraph("p") }},
		{"empty pending", func(d *document.Document) { d.BookmarkNext(""); d.Paragraph("p") }},
		{"unicode", func(d *document.Document) { d.BookmarkNext("ação"); d.Paragraph("p") }},
		{"length", func(d *document.Document) { d.BookmarkNext(strings.Repeat("a", 41)); d.Paragraph("p") }},
		{"duplicate paragraph", func(d *document.Document) {
			d.BookmarkNext("same")
			d.Paragraph("a")
			d.BookmarkNext("same")
			d.Heading("b", 1)
		}},
		{"duplicate across cells", func(d *document.Document) {
			d.AddTable(document.Table{Headers: []string{"H"}, Widths: []int{4000}, Rows: [][]document.Cell{{{Text: "a", Bookmark: "same"}}, {{Text: "b", Bookmark: "same"}}}})
		}},
		{"duplicate cell paragraph", func(d *document.Document) {
			d.BookmarkNext("same")
			d.Paragraph("a")
			d.AddTable(document.Table{Headers: []string{"H"}, Widths: []int{4000}, Rows: [][]document.Cell{{{Text: "b", Bookmark: "same"}}}})
		}},
		{"case duplicate", func(d *document.Document) {
			d.BookmarkNext("CaseTarget")
			d.Paragraph("one")
			d.BookmarkNext("casetarget")
			d.Paragraph("two")
		}},
		{"case duplicate cell paragraph", func(d *document.Document) {
			d.BookmarkNext("CaseTarget")
			d.Paragraph("one")
			d.AddTable(document.Table{Headers: []string{"H"}, Widths: []int{4000}, Rows: [][]document.Cell{{{Text: "two", Bookmark: "casetarget"}}}})
		}},
		{"target spelling remains exact", func(d *document.Document) {
			d.BookmarkNext("CaseTarget")
			d.Paragraph("one")
			d.AddTable(document.Table{Headers: []string{"H"}, Widths: []int{4000}, Rows: [][]document.Cell{{{Text: "link", LinkTarget: "casetarget"}}}})
		}},
		{"dangling", func(d *document.Document) { d.BookmarkNext("later"); d.PageBreak(); d.CodeTable("H", "literal") }},
		{"invalid cell", func(d *document.Document) {
			d.AddTable(document.Table{Headers: []string{"H"}, Widths: []int{4000}, Rows: [][]document.Cell{{{Text: "b", Bookmark: "bad name"}}}})
		}},
		{"unresolved target", func(d *document.Document) {
			d.AddTable(document.Table{Headers: []string{"H"}, Widths: []int{4000}, Rows: [][]document.Cell{{{Text: "b", LinkTarget: "missing"}}}})
		}},
		{"invalid target", func(d *document.Document) {
			d.AddTable(document.Table{Headers: []string{"H"}, Widths: []int{4000}, Rows: [][]document.Cell{{{Text: "b", LinkTarget: "bad\x00"}}}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := fixture(t)
			tc.build(d)
			var b bytes.Buffer
			if err := d.Write(t.Context(), &b); err == nil || b.Len() != 0 {
				t.Fatal("invalid navigation emitted output", err, b.Len())
			}
		})
	}
	d := fixture(t)
	d.BookmarkNext(strings.Repeat("a", 40))
	d.Paragraph("valid")
	var b bytes.Buffer
	if err := d.Write(t.Context(), &b); err != nil {
		t.Fatal("valid boundary name", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	b.Reset()
	if err := d.Write(ctx, &b); !errors.Is(err, context.Canceled) || b.Len() != 0 {
		t.Fatal("cancel/output", err)
	}
}
