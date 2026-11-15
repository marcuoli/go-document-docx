// Command code-table writes synthetic literal source in a multipage code table.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/publication"
	"github.com/marcuoli/go-document-docx/report"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: code-table <output.docx>")
		os.Exit(2)
	}
	d, err := document.New(report.Defaults())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	d.Heading("Literal source example", 1)
	d.Paragraph("Synthetic source is displayed as text. No statement is executed.")
	d.CodeTable("Source — literal Unicode and whitespace", "-- ação 東京 😀 <&>\r\nCREATE TABLE example_items (\n\tid NUMBER,\n\tlabel VARCHAR2(80)\n);\n\n-- blank line above retained\n"+strings.Repeat("long_identifier_", 50))
	var code strings.Builder
	for i := 1; i <= 500; i++ {
		fmt.Fprintf(&code, "-- line %03d: exact source text\nSELECT '%03d — ação' AS label FROM example_items;\n", i, i)
	}
	d.CodeTable("Source — continued code", code.String())
	d.HeadingOnNewPage("Following section", 1)
	d.Paragraph("END OF SYNTHETIC EXAMPLE")
	if _, err := publication.Publish(context.Background(), os.Args[1], false, d); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
