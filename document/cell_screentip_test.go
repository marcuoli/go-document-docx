package document_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
)

func TestCellScreenTipPreservesAppearanceAndInternalTargets(t *testing.T) {
	tip := "Present <&> \"quoted\" ação\nnext"
	build := func(value string) []byte {
		d := fixture(t)
		for range 2 {
			rows := [][]document.Cell{{
				{Text: "✔", ScreenTip: value, Bold: true, TextColor: "000000", Alignment: "right"},
				{ScreenTip: value, Alignment: "center", TextColor: "007F00", Tone: document.Difference, Runs: []document.CellRun{{Text: "✘\r", TextColor: "B3261E", Italic: true}, {}, {Text: "\n✔ ⚠️\t<&>"}}},
			}}
			d.AddTable(document.Table{Headers: []string{"One", "Two"}, Widths: []int{2500, 2500}, FontSize: 14, Rows: rows})
			rows[0][0].ScreenTip = "mutated after AddTable"
			rows[0][1].ScreenTip = "mutated after AddTable"
		}
		d.CodeTable("Literal code", "  value := 42\n\tprint(value)")
		var out bytes.Buffer
		if err := d.Write(t.Context(), &out); err != nil {
			t.Fatal(err)
		}
		return out.Bytes()
	}
	baseline, withTip := parts(t, build("")), parts(t, build(tip))
	if strings.Contains(baseline["word/document.xml"], "w:hyperlink") || strings.Contains(baseline["word/document.xml"], "w:bookmark") {
		t.Fatal("empty ScreenTip added link markup")
	}
	wrapper := regexp.MustCompile(`<w:bookmark(?:Start|End)\b[^>]*/>|</?w:hyperlink\b[^>]*>`)
	for name, before := range baseline {
		after := withTip[name]
		if name == "word/document.xml" {
			after = wrapper.ReplaceAllString(after, "")
		}
		if after != before {
			t.Fatalf("ScreenTip changed appearance/content/package part %s", name)
		}
	}
	decoder := xml.NewDecoder(strings.NewReader(withTip["word/document.xml"]))
	ids, names, ends, anchors := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	links := 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range start.Attr {
			attrs[attr.Name.Local] = attr.Value
		}
		switch start.Name.Local {
		case "bookmarkStart":
			if ids[attrs["id"]] || names[attrs["name"]] || attrs["name"] == "" {
				t.Fatal("missing or duplicated bookmark", attrs)
			}
			ids[attrs["id"]], names[attrs["name"]] = true, true
		case "bookmarkEnd":
			if ends[attrs["id"]] {
				t.Fatal("duplicate bookmark end")
			}
			ends[attrs["id"]] = true
		case "hyperlink":
			if attrs["tooltip"] != tip || attrs["anchor"] == "" || attrs["id"] != "" || attrs["history"] != "0" {
				t.Fatal("tooltip changed or acquired external target", attrs)
			}
			anchors[attrs["anchor"]] = true
			links++
		}
	}
	if links != 4 || len(ids) != 4 || len(anchors) != 4 || len(ends) != 4 {
		t.Fatal("missing/duplicated internal targets", links, ids, ends, anchors)
	}
	for id := range ids {
		if !ends[id] {
			t.Fatal("unpaired bookmark", id)
		}
	}
	for anchor := range anchors {
		if !names[anchor] {
			t.Fatal("missing self anchor", anchor)
		}
	}
}

func TestCellScreenTipValidationAndErrors(t *testing.T) {
	for _, tc := range []struct{ name, tip string }{
		{"XML", "invalid\x00"}, {"UTF8", "\xff"}, {"aggregate text", strings.Repeat("a", 64<<20)},
		{"expanded XML", strings.Repeat("&", 27<<20)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := fixture(t)
			d.AddTable(document.Table{Headers: []string{"Value"}, Widths: []int{5000}, Rows: [][]document.Cell{{{Text: "✔", ScreenTip: tc.tip}}}})
			var out bytes.Buffer
			if err := d.Write(t.Context(), &out); err == nil || out.Len() != 0 {
				t.Fatalf("invalid tooltip accepted/output emitted: %v/%d", err, out.Len())
			}
		})
	}
	d := fixture(t)
	d.AddTable(document.Table{Headers: []string{"Value"}, Widths: []int{5000}, Rows: [][]document.Cell{{{Text: "✔", ScreenTip: "Present"}}}})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var out bytes.Buffer
	if err := d.Write(ctx, &out); !errors.Is(err, context.Canceled) || out.Len() != 0 {
		t.Fatal("cancellation/output changed", err, out.Len())
	}
	sentinel := errors.New("writer failed")
	if err := d.Write(t.Context(), screenTipFailureWriter{sentinel}); !errors.Is(err, sentinel) {
		t.Fatal("writer error identity lost", err)
	}
}

type screenTipFailureWriter struct{ err error }

func (w screenTipFailureWriter) Write([]byte) (int, error) { return 0, w.err }
