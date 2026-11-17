package htmlcontent

import (
	"bytes"
	"context"
	"github.com/marcuoli/go-document-docx/internal/opc"
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

func TestIndependentHTMLListsRestartAndResumeOuterSequence(t *testing.T) {
	d, err := editing.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	markup := `<ol><li>Outer first<ol><li>Nested first</li></ol><ol><li>Nested restart</li></ol></li><li>Outer second</li></ol><p>Between</p><ol start="1"><li>Restart</li></ol>`
	if err := Append(t.Context(), d, strings.NewReader(markup), Options{}); err != nil {
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
	ids := map[string]string{}
	levels := map[string]string{}
	for _, p := range opc.Elements(doc.Root(), opc.Word, "p") {
		pr := opc.Child(p, opc.Word, "pPr")
		if pr == nil {
			continue
		}
		num := opc.Child(pr, opc.Word, "numPr")
		if num == nil {
			continue
		}
		var text strings.Builder
		for _, n := range opc.Elements(p, opc.Word, "t") {
			text.WriteString(n.Text())
		}
		ids[text.String()] = opc.Attr(opc.Child(num, opc.Word, "numId"), opc.Word, "val")
		levels[text.String()] = opc.Attr(opc.Child(num, opc.Word, "ilvl"), opc.Word, "val")
	}
	outer := ids["Outer first"]
	if len(ids) != 5 || outer == "" || ids["Outer second"] != outer || ids["Restart"] == outer || ids["Nested first"] == ids["Nested restart"] || ids["Nested first"] == outer || ids["Nested restart"] == ids["Restart"] {
		t.Fatal("list boundaries/outer continuation lost", ids)
	}
	if levels["Nested first"] != "1" || levels["Nested restart"] != "1" || levels["Restart"] != "0" {
		t.Fatal("list nesting lost", levels)
	}
	numbering, err := pkg.XML("word/numbering.xml")
	if err != nil {
		t.Fatal(err)
	}
	for text, level := range map[string]string{"Outer first": "0", "Nested first": "1", "Nested restart": "1", "Restart": "0"} {
		found := false
		for _, num := range opc.Elements(numbering.Root(), opc.Word, "num") {
			if opc.Attr(num, opc.Word, "numId") != ids[text] {
				continue
			}
			override := opc.Child(num, opc.Word, "lvlOverride")
			if override != nil && opc.Attr(override, opc.Word, "ilvl") == level && opc.Attr(opc.Child(override, opc.Word, "startOverride"), opc.Word, "val") == "1" {
				found = true
			}
		}
		if !found {
			t.Fatal("independent list lacks explicit restart", text)
		}
	}
}
