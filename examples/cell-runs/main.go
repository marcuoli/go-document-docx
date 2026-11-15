// Command cell-runs writes a synthetic table with independently colored spans.
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
		fmt.Fprintln(os.Stderr, "usage: cell-runs <output.docx>")
		os.Exit(2)
	}
	d, err := document.New(report.Defaults())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	d.Heading("Literal cell text runs", 1)
	d.Paragraph("Each span has a caller-selected color. Literal labels never determine their formatting.")
	d.AddTable(document.Table{Headers: []string{"Property", "Members"}, Widths: []int{3000, 6746}, Rows: [][]document.Cell{
		{{Text: "Mixed literal labels"}, {Runs: []document.CellRun{{Text: "1:N/A\n"}, {Text: "2:N/A", TextColor: "B3261E"}}}},
		{{Text: "Unicode and indentation"}, {Bold: true, Runs: []document.CellRun{{Text: "  ação 東京 😀\n\t"}, {Text: "Literal <&>", TextColor: "1565C0"}}}},
		{{Text: "Legacy whole cell"}, {Text: "N/A", TextColor: "007F00"}},
	}})
	if _, err := publication.Publish(context.Background(), os.Args[1], false, d); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
