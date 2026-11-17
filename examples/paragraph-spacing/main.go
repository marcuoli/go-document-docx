// Command paragraph-spacing writes a synthetic document with selected heading spacing.
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
		fmt.Fprintln(os.Stderr, "usage: paragraph-spacing <output.docx>")
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
	d.Title("Selected heading spacing")
	d.Paragraph("Literal prose remains unchanged. Heading space is a paragraph property rather than an added blank paragraph.")
	d.Heading("First level", 1)
	d.Paragraph("A major section.")
	d.Heading("Second level", 2)
	d.Paragraph("A related item.")
	var b bytes.Buffer
	if err := d.Write(ctx, &b); err != nil {
		return err
	}
	spaced, err := formatting.SetParagraphSpaceBefore(ctx, b.Bytes(), 720, "Heading1")
	if err != nil {
		return err
	}
	spaced, err = formatting.SetParagraphSpaceBefore(ctx, spaced, 480, "Heading2")
	if err != nil {
		return err
	}
	_, err = publication.PublishSource(ctx, path, false, publication.FromBytes(spaced))
	return err
}
