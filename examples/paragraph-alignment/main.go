// Command paragraph-alignment writes a synthetic example with justified prose.
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/formatting"
	"github.com/marcuoli/go-document-docx/publication"
	"github.com/marcuoli/go-document-docx/report"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: paragraph-alignment <output.docx>")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string) error {
	ctx := context.Background()
	d, err := document.New(report.Defaults())
	if err != nil {
		return err
	}
	d.Title("Paragraph alignment example")
	d.Paragraph("This synthetic document demonstrates justified prose while preserving headings, exact values and table formatting. The final line of each paragraph keeps the normal reader behavior rather than stretching short lines across the page.")
	d.Heading("Selected prose and protected table data", 1)
	d.ParagraphWithNext("Paragraphs can remain with the following table while using the requested alignment. Alignment changes presentation without changing the text, numerical values or other paragraph styles.")
	d.AddTable(document.Table{Headers: []string{"Property", "Value"}, Widths: []int{4000, 5746}, Rows: [][]document.Cell{{{Text: "Exact count"}, {Text: "9007199254740993"}}}})
	var b bytes.Buffer
	if err := d.Write(ctx, &b); err != nil {
		return err
	}
	aligned, err := formatting.AlignParagraphs(ctx, b.Bytes(), "justify", "", "Normal", "Lead")
	if err != nil {
		return err
	}
	_, err = publication.PublishSource(ctx, path, false, publication.FromBytes(aligned))
	return err
}
