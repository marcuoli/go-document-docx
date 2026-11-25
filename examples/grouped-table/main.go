// Command grouped-table writes a synthetic comparison with grouped headings.
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
		fmt.Fprintln(os.Stderr, "usage: grouped-table <output.docx>")
		os.Exit(2)
	}
	if err := generate(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(path string) error {
	options := report.Defaults()
	options.TableStyle = &document.TableStyle{
		GridBorders: true, BorderColor: "8C939D", HeaderFill: "EFF2F6",
		PaddingTop: 65, PaddingBottom: 65, PaddingLeft: 90, PaddingRight: 90,
	}
	d, err := document.New(options)
	if err != nil {
		return err
	}
	d.AddTable(document.Table{
		Widths: []int{2050, 1500, 1500, 1500, 1500, 950},
		HeaderRows: [][]document.Cell{
			{{}, {Text: "DB_PRIMARY", ColSpan: 2, Alignment: "center"}, {Text: "DB_STANDBY", ColSpan: 2, Alignment: "center"}, {}},
			{{Text: "PROPERTY"}, {Text: "DBTIMEZONE"}, {Text: "TIME"}, {Text: "DBTIMEZONE"}, {Text: "TIME"}, {Text: "Result"}},
		},
		Rows: [][]document.Cell{
			{{Text: "DBTIMEZONE"}, {Text: "+00:00", ColSpan: 2}, {Text: "+00:00", ColSpan: 2}, {Text: "✓", TextColor: "008000"}},
			{{Text: "SESSION CLOCK"}, {Text: "+00:00"}, {Text: "13:00:00"}, {Text: "+00:00"}, {Text: "13:00:00"}, {Text: "✓", TextColor: "008000"}},
		},
	})
	_, err = publication.Publish(context.Background(), path, false, d)
	return err
}
