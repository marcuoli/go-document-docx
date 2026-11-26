// Command logo inserts an image in the first table cell of an existing document.
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/publication"
	"github.com/marcuoli/go-document-docx/report"
)

func main() {
	input := flag.String("input", "", "existing DOCX with a table")
	image := flag.String("logo", "", "PNG or JPEG artwork file")
	output := flag.String("output", "", "new DOCX file")
	flag.Parse()
	if *input == "" || *image == "" || *output == "" {
		flag.Usage()
		os.Exit(2)
	}
	file, err := os.Open(*image)
	if err != nil {
		log.Fatal(err)
	}
	logo, err := editing.ReadLogo(file)
	closeErr := file.Close()
	if err != nil {
		log.Fatal(err)
	}
	if closeErr != nil {
		log.Fatal(closeErr)
	}
	source, err := os.ReadFile(*input)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	doc, err := logo.CenterInTableCell(ctx, source, report.Defaults(), editing.LogoCell{
		SideTwips:        1700,
		SpaceBeforeTwips: 100,
	})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := publication.Save(ctx, *output, false, publication.FromBytes(doc)); err != nil {
		log.Fatal(err)
	}
}
