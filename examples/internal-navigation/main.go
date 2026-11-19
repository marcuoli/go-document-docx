// Command internal-navigation writes a synthetic literal navigation example.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/publication"
	"github.com/marcuoli/go-document-docx/report"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: internal-navigation <output.docx>")
		os.Exit(2)
	}
	d, err := document.New(report.Defaults())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	d.Heading("Internal navigation", 1)
	d.Paragraph("Each section link resolves to the exact named paragraph or table cell. Tooltips remain literal.")
	d.AddTable(document.Table{Headers: []string{"Destination", "Description"}, Widths: []int{3500, 6246}, Rows: [][]document.Cell{
		{{Text: "Detail heading", LinkTarget: "detail_heading", ScreenTip: "Jump to the detail heading"}, {Text: "Forward reference to a paragraph"}},
		{{Text: "Exact table cell", LinkTarget: "detail_cell"}, {Text: "Forward reference to one cell"}},
	}})
	d.BookmarkNext("detail_heading")
	d.HeadingOnNewPage("Details", 1)
	d.Paragraph("Navigation preserves the shared report typography and literal Unicode text.")
	d.AddTable(document.Table{Headers: []string{"Item", "Exact detail"}, Widths: []int{3500, 6246}, Rows: [][]document.Cell{
		{{Text: "Synthetic item"}, {Bookmark: "detail_cell", Runs: []document.CellRun{{Text: "Exact <&> detail", Italic: true}, {Text: "\nSecond line", TextColor: "B3261E"}}}},
	}})
	if _, err := publication.Publish(context.Background(), os.Args[1], false, d); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
