package document

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestNavigationPreservesEachRun(t *testing.T) {
	for _, color := range []string{"", "007F00"} {
		t.Run("cell_color_"+color, func(t *testing.T) {
			cell := Cell{LinkTarget: "target", ScreenTip: "literal <&>", Bold: true, Runs: []CellRun{
				{Text: "before\r", Italic: true}, {Text: "", Italic: true}, {Text: "\nação\t<&>", TextColor: "B3261E"}, {Text: " after"},
			}}
			var b strings.Builder
			if err := writeCellParagraph(t.Context(), &b, "TableText", cell, color, 14, &bookmarkState{}); err != nil {
				t.Fatal(err)
			}
			var paragraph struct {
				Links []struct {
					Anchor string     `xml:"anchor,attr"`
					Tip    string     `xml:"tooltip,attr"`
					Nested []struct{} `xml:"hyperlink"`
					Runs   []struct {
						Props struct {
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
							Underline struct {
								Value string `xml:"val,attr"`
							} `xml:"u"`
						} `xml:"rPr"`
						Text   []string   `xml:"t"`
						Breaks []struct{} `xml:"br"`
						Tabs   []struct{} `xml:"tab"`
					} `xml:"r"`
				} `xml:"hyperlink"`
			}
			if err := xml.Unmarshal([]byte(b.String()), &paragraph); err != nil {
				t.Fatal(err)
			}
			if len(paragraph.Links) != 1 {
				t.Fatal("wrong link count", b.String())
			}
			link := paragraph.Links[0]
			if link.Anchor != "target" || link.Tip != "literal <&>" || len(link.Nested) != 0 || len(link.Runs) != 4 {
				t.Fatal("link structure changed", b.String())
			}
			inherited := color
			if inherited == "" {
				inherited = "315357"
			}
			for i, run := range link.Runs {
				wantColor := inherited
				if i == 2 {
					wantColor = "B3261E"
				}
				if run.Props.Bold == nil || (run.Props.Italic != nil) != (i < 2) || run.Props.Color.Value != wantColor || run.Props.Size.Value != 14 || run.Props.ComplexSize.Value != 14 || run.Props.Underline.Value != "single" {
					t.Fatalf("run%d formatting changed: %+v", i, run.Props)
				}
				wantBreaks, wantTabs := 0, 0
				if i == 0 {
					wantBreaks = 1
				}
				if i == 2 {
					wantTabs = 1
				}
				if strings.Join(run.Text, "") != []string{"before", "", "ação<&>", " after"}[i] || len(run.Breaks) != wantBreaks || len(run.Tabs) != wantTabs {
					t.Fatalf("run%d literal content changed: %+v", i, run)
				}
			}
		})
	}
}
