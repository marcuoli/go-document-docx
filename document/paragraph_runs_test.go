package document_test

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
)

func TestParagraphRunsOwnTextAndColor(t *testing.T) {
	d := fixture(t)
	runs := []document.ParagraphRun{
		{Text: "Validation <&> passed. "},
		{Text: "✓", TextColor: "008000"},
	}
	d.BookmarkNext("validation")
	d.ParagraphRuns(runs...)
	runs[0].Text = "changed"
	runs[1].TextColor = "not a color"

	data := paragraphRunsDOCX(t, d)
	main := parts(t, data)["word/document.xml"]
	var doc struct {
		Paragraphs []struct {
			Style struct {
				Value string `xml:"val,attr"`
			} `xml:"pPr>pStyle"`
			Runs []struct {
				Color *struct {
					Value string `xml:"val,attr"`
				} `xml:"rPr>color"`
				Text string `xml:"t"`
			} `xml:"r"`
		} `xml:"body>p"`
	}
	if err := xml.Unmarshal([]byte(main), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Paragraphs) != 1 || doc.Paragraphs[0].Style.Value != "Normal" || len(doc.Paragraphs[0].Runs) != 2 {
		t.Fatalf("want one paragraph with two runs, got %+v", doc.Paragraphs)
	}
	plain, mark := doc.Paragraphs[0].Runs[0], doc.Paragraphs[0].Runs[1]
	if plain.Text != "Validation <&> passed. " || plain.Color != nil {
		t.Fatalf("prose changed or acquired a color: %+v", plain)
	}
	if mark.Text != "✓" || mark.Color == nil || mark.Color.Value != "008000" {
		t.Fatalf("checkmark has wrong text or color: %+v", mark)
	}
	if nav := inspectNavigation(t, data); nav.Text["validation"] != "Validation <&> passed. ✓" {
		t.Fatalf("bookmark missed paragraph content: %q", nav.Text["validation"])
	}
}

func TestParagraphRunsLiteralBreaksAndPlainParity(t *testing.T) {
	d := fixture(t)
	d.ParagraphRuns(
		document.ParagraphRun{Text: "<a>\r"},
		document.ParagraphRun{},
		document.ParagraphRun{Text: "\n\t ação", TextColor: "fF0080"},
	)
	if got := codeRunText(t, parts(t, paragraphRunsDOCX(t, d))["word/document.xml"]); got != "<a>\n\t ação" {
		t.Fatalf("literal run text changed: %q", got)
	}

	plain, rich := fixture(t), fixture(t)
	plain.Paragraph("Inherited \r\n\t<&>")
	plain.Paragraph("")
	plain.ParagraphWithNext("Keep with next")
	rich.ParagraphRuns(document.ParagraphRun{Text: "Inherited \r\n\t<&>"})
	rich.ParagraphRuns()
	rich.ParagraphWithNext("Keep with next")
	if !bytes.Equal(paragraphRunsDOCX(t, plain), paragraphRunsDOCX(t, rich)) {
		t.Fatal("unstyled and empty run paragraphs must retain plain paragraph output")
	}
}

func TestParagraphRunsRejectBeforeOutput(t *testing.T) {
	for _, test := range []struct {
		name string
		run  document.ParagraphRun
	}{
		{"hash prefix", document.ParagraphRun{Text: "x", TextColor: "#FF0000"}},
		{"nonhex", document.ParagraphRun{Text: "x", TextColor: "GG0000"}},
		{"invalid UTF-8", document.ParagraphRun{Text: "\xff"}},
		{"invalid XML", document.ParagraphRun{Text: "x\x00"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := fixture(t)
			d.Paragraph("Keep valid text")
			d.ParagraphRuns(document.ParagraphRun{Text: "first span"}, test.run)
			var out bytes.Buffer
			if err := d.Write(t.Context(), &out); err == nil || out.Len() != 0 {
				t.Fatalf("invalid runs reached output: %v, %d bytes", err, out.Len())
			}
		})
	}
	d := fixture(t)
	large := strings.Repeat("x", 32<<20)
	d.ParagraphRuns(document.ParagraphRun{Text: large}, document.ParagraphRun{Text: large})
	var out bytes.Buffer
	if err := d.Write(t.Context(), &out); err == nil || !strings.Contains(err.Error(), "64 MiB") || out.Len() != 0 {
		t.Fatalf("run text escaped aggregate budget: %v, %d bytes", err, out.Len())
	}
}

func paragraphRunsDOCX(t *testing.T, d *document.Document) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := d.Write(t.Context(), &b); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
