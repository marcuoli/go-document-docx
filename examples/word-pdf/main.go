// Command word-pdf exports a synthetic Word document as a PDF.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/publication"
	"github.com/marcuoli/go-document-docx/report"
	"github.com/marcuoli/go-document-docx/word"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: word-pdf <output.pdf>")
		os.Exit(2)
	}
	if err := export(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func export(path string) error {
	doc, err := document.New(report.Defaults())
	if err != nil {
		return err
	}
	doc.Heading("First synthetic section", 1)
	doc.Paragraph("Word lays out the first page.")
	doc.HeadingOnNewPage("Second synthetic section", 1)
	doc.ParagraphRuns(
		document.ParagraphRun{Text: "Validation passed. "},
		document.ParagraphRun{Text: "✓", TextColor: "008000"},
	)
	_, err = publication.Save(context.Background(), path, false, word.PDF(doc, word.Options{}))
	return err
}
