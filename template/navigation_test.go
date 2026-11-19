package template

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	word "github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/internal/opc"
	"github.com/marcuoli/go-document-docx/report"
)

func literalNavigationPage(t *testing.T, label string) []byte {
	t.Helper()
	d, err := word.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	d.AddTable(word.Table{Headers: []string{"Link"}, Widths: []int{5000}, Rows: [][]word.Cell{
		{{Text: label + " heading link", LinkTarget: "scope", ScreenTip: "Navigate " + label}},
		{{Text: label + " long link", LinkTarget: strings.Repeat("a", 40)}},
	}})
	if label == "First" {
		for _, name := range []string{"Closing1_scope", "Closing1_bookmark_6", "Closing1_bookmark_7"} {
			d.BookmarkNext(name)
			d.Paragraph("Reserved " + name)
		}
	}
	d.BookmarkNext("scope")
	d.Heading(label+" heading", 1)
	d.AddTable(word.Table{Headers: []string{"Detail"}, Widths: []int{5000}, Rows: [][]word.Cell{{{Text: label + " exact cell", Bookmark: strings.Repeat("a", 40)}}}})
	var b bytes.Buffer
	if err := d.Write(t.Context(), &b); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestLiteralNavigationCompositionKeepsPreciseCollidingDestinations(t *testing.T) {
	first, last := literalNavigationPage(t, "First"), literalNavigationPage(t, "Imported")
	beforeFirst, beforeLast := bytes.Clone(first), bytes.Clone(last)
	combined, err := Compose(t.Context(), first, last)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, beforeFirst) || !bytes.Equal(last, beforeLast) {
		t.Fatal("composition changed caller inputs")
	}
	p, err := opc.Read(t.Context(), combined)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := p.XML("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	names, ids := map[string]string{}, map[string]bool{}
	for _, mark := range opc.Elements(doc.Root(), opc.Word, "bookmarkStart") {
		name, id := opc.Attr(mark, opc.Word, "name"), opc.Attr(mark, opc.Word, "id")
		if _, exists := names[name]; exists || ids[id] || len(name) > 40 {
			t.Fatal("colliding/invalid bookmark", name, id)
		}
		ids[id] = true
		var value strings.Builder
		for _, node := range opc.Elements(mark.Parent(), opc.Word, "t") {
			value.WriteString(node.Text())
		}
		names[name] = value.String()
	}
	expected := map[string]string{
		"First heading link": "First heading", "Imported heading link": "Imported heading",
		"First long link": "First exact cell", "Imported long link": "Imported exact cell",
	}
	seen := map[string]bool{}
	for _, link := range opc.Elements(doc.Root(), opc.Word, "hyperlink") {
		var label strings.Builder
		for _, node := range opc.Elements(link, opc.Word, "t") {
			label.WriteString(node.Text())
		}
		target := opc.Attr(link, opc.Word, "anchor")
		if names[target] != expected[label.String()] || expected[label.String()] == "" {
			t.Fatal("link retargeted during composition", label.String(), target, names)
		}
		for parent := link.Parent(); parent != nil; parent = parent.Parent() {
			if parent.Tag == "hyperlink" && parent.NamespaceURI() == opc.Word {
				t.Fatal("nested link")
			}
		}
		seen[label.String()] = true
	}
	if len(seen) != len(expected) || len(names) != 7 {
		t.Fatal("missing links/destinations", seen, names)
	}
}

// Count cancellation checkpoints as a deterministic work bound, rather than
// asserting wall-clock timing on a busy host.
type composeProbeContext struct {
	context.Context
	calls int
}

func (c *composeProbeContext) Err() error { c.calls++; return c.Context.Err() }
func TestBookmarkFallbackSearchAdvancesAcrossImports(t *testing.T) {
	const count = 160
	page := func(imported bool) []byte {
		d, err := word.New(report.Defaults())
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("long_%035d", i)
			d.BookmarkNext(name)
			d.Paragraph("long")
		}
		if !imported {
			for i := 3*count + 1; i <= 5*count; i++ {
				d.BookmarkNext(fmt.Sprintf("Closing1_bookmark_%d", i))
				d.Paragraph("reserved")
			}
		}
		var b bytes.Buffer
		if err := d.Write(t.Context(), &b); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	ctx := &composeProbeContext{Context: t.Context()}
	if _, err := Compose(ctx, page(false), page(true)); err != nil {
		t.Fatal(err)
	}
	// Admission, XML serialization and all name probes fit comfortably below
	// this linear budget; rescanning the occupied range needs >38,000 probes.
	if ctx.calls > count*100 {
		t.Fatalf("fallback search repeated occupied candidates: %d checkpoints", ctx.calls)
	}
	t.Logf("%d bookmarks: %d cancellation checkpoints", count, ctx.calls)
}
