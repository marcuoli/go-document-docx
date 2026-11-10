package htmlcontent

import (
	"context"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/report"
)

func TestHeaderlessSelectionPreservesFirstEvidenceRow(t *testing.T) {
	markup := "<table id='evidence'><tr><td>first</td><td>9007199254740993</td></tr><tr><td>second</td><td>null</td></tr></table>"
	result, err := ReadTable(context.Background(), strings.NewReader(markup), Selector{ID: "evidence"}, TableOptions{HeaderlessHeaders: []string{"Name", "Value"}, Select: []int{1, 0}, OverrideHeaders: []string{"Amount", "Identity"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Data.Rows) != 2 || result.Data.Rows[0][0] != "9007199254740993" || result.Data.Rows[1][0] != "null" {
		t.Fatal(result)
	}
}
func TestAmbiguousAndRaggedTablesFailWithoutFallback(t *testing.T) {
	for _, markup := range []string{"<table><tr><th>A</th></tr><tr><td>x</td><td>y</td></tr></table>", "<table><tr><th colspan='2'>A</th></tr></table>", "<table><tr><th>A</th></tr></table><table><tr><th>B</th></tr></table>"} {
		if _, err := ReadTable(context.Background(), strings.NewReader(markup), Selector{}, TableOptions{}); err == nil {
			t.Fatal(markup)
		}
	}
}
func TestImportLiteralMarkupAndLists(t *testing.T) {
	d, err := editing.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	input := "<h1>Review</h1><p id='scope'>Exact &amp;lt;value&amp;gt; <b>bold</b><br>next <a href='#scope'>scope</a></p><ol><li>First<ul><li>Nested</li></ul></li><li>Second</li></ol><script>secret()</script>"
	if err := Append(context.Background(), d, strings.NewReader(input), Options{}); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	if err := d.Write(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range d.Native().Paragraphs() {
		if strings.Contains(p.Text(), "&lt;value&gt;") {
			found = true
		}
		if strings.Contains(p.Text(), "secret()") {
			t.Fatal("script imported")
		}
	}
	if !found {
		t.Fatal("HTML entities decoded twice")
	}
}

func TestNamedSelectionKeepsColorsAndRejectsInvalidMetadata(t *testing.T) {
	markup := "<table><tr><th>Name</th><th>Value</th></tr><tr><td bgcolor='#FFF0D5'>first</td><td color='315357'>9007199254740993</td></tr></table>"
	parsed, err := ReadTable(context.Background(), strings.NewReader(markup), Selector{}, TableOptions{SelectNames: []string{"Value", "Name"}})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Data.Rows[0][0] != "9007199254740993" || parsed.Colors[[2]int{1, 0}].Foreground != "315357" || parsed.Colors[[2]int{1, 1}].Background != "FFF0D5" {
		t.Fatal(parsed)
	}
	if _, err := ReadTable(context.Background(), strings.NewReader(strings.ReplaceAll(markup, "#FFF0D5", "not-a-color")), Selector{}, TableOptions{}); err == nil {
		t.Fatal("invalid metadata silently ignored")
	}
}
