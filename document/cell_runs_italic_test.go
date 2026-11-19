package document

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestCellRunItalicPreservesSpanFormatting(t *testing.T) {
	runs := []CellRun{
		{Text: "before\r"},
		{Italic: true, TextColor: "1565C0"},
		{Text: "\nação\t<&>", TextColor: "B3261E", Italic: true},
		{Text: " after"},
	}
	d := &Document{}
	rows := [][]Cell{{{Runs: runs, Bold: true, TextColor: "007F00", Alignment: "right"}}}
	d.AddTable(Table{Headers: []string{"Value"}, Widths: []int{5000}, FontSize: 14, Rows: rows})
	// The supplied run slice is owned: caller edits cannot add or remove italics.
	runs[0].Italic = true
	runs[2].Italic = false
	owned := d.blocks[0].table
	var out strings.Builder
	if err := writeCellParagraph(t.Context(), &out, "TableText", owned.Rows[0][0], "007F00", owned.FontSize, 0); err != nil {
		t.Fatal(err)
	}
	var paragraph struct {
		Properties struct {
			Alignment struct {
				Value string `xml:"val,attr"`
			} `xml:"jc"`
		} `xml:"pPr"`
		Runs []struct {
			Properties struct {
				Bold   *struct{} `xml:"b"`
				Italic *struct{} `xml:"i"`
				Color  struct {
					Value string `xml:"val,attr"`
				} `xml:"color"`
				Size struct {
					Value int `xml:"val,attr"`
				} `xml:"sz"`
				ComplexSize struct {
					Value int `xml:"val,attr"`
				} `xml:"szCs"`
			} `xml:"rPr"`
			Text   []string   `xml:"t"`
			Breaks []struct{} `xml:"br"`
			Tabs   []struct{} `xml:"tab"`
		} `xml:"r"`
	}
	if err := xml.Unmarshal([]byte(out.String()), &paragraph); err != nil {
		t.Fatal(err)
	}
	if paragraph.Properties.Alignment.Value != "right" || len(paragraph.Runs) != 4 {
		t.Fatal("paragraph alignment/run count changed", out.String())
	}
	for i, run := range paragraph.Runs {
		wantItalic := i == 1 || i == 2
		if (run.Properties.Italic != nil) != wantItalic {
			t.Fatalf("run%d italic changed: %+v", i, run.Properties)
		}
		if run.Properties.Bold == nil || run.Properties.Color.Value != []string{"007F00", "1565C0", "B3261E", "007F00"}[i] || run.Properties.Size.Value != 14 || run.Properties.ComplexSize.Value != 14 {
			t.Fatalf("run%d formatting changed: %+v", i, run.Properties)
		}
	}
	if strings.Join(paragraph.Runs[0].Text, "") != "before" || len(paragraph.Runs[0].Breaks) != 1 || len(paragraph.Runs[2].Breaks) != 0 || len(paragraph.Runs[2].Tabs) != 1 || strings.Join(paragraph.Runs[2].Text, "") != "ação<&>" || strings.Join(paragraph.Runs[3].Text, "") != " after" {
		t.Fatal("literal text, cross-span CRLF, tab or Unicode changed", out.String())
	}
	if !strings.Contains(out.String(), `<w:b/><w:i/><w:color w:val="B3261E"/><w:sz w:val="14"/><w:szCs w:val="14"/>`) {
		t.Fatal("OOXML property order changed", out.String())
	}
}

func TestRunItalicNativeAndInheritedSerialization(t *testing.T) {
	cases := []struct {
		name         string
		italic, bold bool
		color        string
		size         int
		want         string
	}{
		{name: "inherit", want: `<w:r><w:t xml:space="preserve">Literal</w:t></w:r>`},
		{name: "italic only", italic: true, want: `<w:r><w:rPr><w:i/></w:rPr><w:t xml:space="preserve">Literal</w:t></w:r>`},
		{name: "legacy formatted", bold: true, color: "007F00", size: 14, want: `<w:r><w:rPr><w:b/><w:color w:val="007F00"/><w:sz w:val="14"/><w:szCs w:val="14"/></w:rPr><w:t xml:space="preserve">Literal</w:t></w:r>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := writeRun(t.Context(), &out, "Literal", tc.color, tc.bold, tc.italic, tc.size, false); err != nil {
				t.Fatal(err)
			}
			if out.String() != tc.want {
				t.Fatalf("serialized run\ngot %s\nwant%s", out.String(), tc.want)
			}
		})
	}
}
