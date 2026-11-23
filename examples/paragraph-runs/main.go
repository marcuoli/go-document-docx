// Command paragraph-runs writes a synthetic report with a colored checkmark.
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
		fmt.Fprintln(os.Stderr, "usage: paragraph-runs <output.docx>")
		os.Exit(2)
	}
	d, err := document.New(report.Defaults())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	d.Paragraph("Validation passed.")
	d.ParagraphRuns(
		document.ParagraphRun{Text: "Validation passed. "},
		document.ParagraphRun{Text: "✓", TextColor: "008000"},
	)
	if _, err := publication.Publish(context.Background(), os.Args[1], false, d); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
