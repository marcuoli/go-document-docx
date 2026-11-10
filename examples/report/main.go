// Command report writes a synthetic technical report to an explicit destination.
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
		fmt.Fprintln(os.Stderr, "usage: report <output.docx>")
		os.Exit(2)
	}
	d, err := document.New(report.Defaults())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	d.Title("Technical Configuration Review")
	d.Subtitle("Synthetic report example")
	d.Paragraph("This report compares two synthetic environments and records findings requiring review.")
	d.PageBreak()
	d.Heading("Executive summary", 1)
	d.Paragraph("One configuration difference requires investigation. No operational acceptance is implied.")
	d.Heading("Environment comparison", 1)
	d.AddTable(document.Table{Headers: []string{"Property", "Source", "Target"}, Widths: []int{3500, 3000, 3246}, Rows: [][]document.Cell{{{Text: "Character set"}, {Text: "AL32UTF8"}, {Text: "AL32UTF8"}}, {{Text: "Rows"}, {Text: "9007199254740993"}, {Text: "9007199254740994", Tone: document.Difference}}}})
	d.Heading("Findings and pending actions", 1)
	d.Paragraph("Review the row-count difference against the agreed cutoff before accepting the destination.")
	if _, err = publication.Publish(context.Background(), os.Args[1], false, d); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
