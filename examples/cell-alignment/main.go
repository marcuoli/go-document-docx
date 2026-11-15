// Command cell-alignment writes caller-aligned literal table cells.
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
		fmt.Fprintln(os.Stderr, "usage: cell-alignment <output.docx>")
		os.Exit(2)
	}
	d, err := document.New(report.Defaults())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	d.Heading("Literal cell alignment", 1)
	d.Paragraph("Callers choose alignment and supply all formatting. Text is never interpreted as a number.")
	d.AddTable(document.Table{Headers: []string{"Alignment", "Literal value"}, Widths: []int{3500, 6246}, Rows: [][]document.Cell{
		{{Text: "Inherited"}, {Text: "1,234.50"}},
		{{Text: "Right"}, {Text: "1,234.50", Alignment: "right"}},
		{{Text: "Centered runs"}, {Alignment: "center", Bold: true, Runs: []document.CellRun{{Text: "10.00\n"}, {Text: "0.0008", TextColor: "B3261E"}}}},
		{{Text: "Justified prose"}, {Text: "Literal words remain intact while the reader lays out this caller-selected paragraph alignment.", Alignment: "justify"}},
	}})
	d.CodeTable("Unchanged code alignment", "  value := 42\n\tprint(value)")
	if _, err := publication.Publish(context.Background(), os.Args[1], false, d); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
