// Command authoring builds a synthetic document using the extended packages.
// Supply an output path in an existing directory; existing files are preserved.
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/htmlcontent"
	"github.com/marcuoli/go-document-docx/publication"
	"github.com/marcuoli/go-document-docx/report"
	"github.com/marcuoli/go-document-docx/report/actions"
	"github.com/marcuoli/go-document-docx/tabledata"
	doctemplate "github.com/marcuoli/go-document-docx/template"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: authoring <output.docx>")
		os.Exit(2)
	}
	if err := run(context.Background(), os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(ctx context.Context, path string) error {
	options := report.Defaults()
	options.Header = "Document authoring reference"
	options.Footer = "Synthetic example"
	doc, err := editing.New(options)
	if err != nil {
		return err
	}
	if _, err := doc.Paragraph("Document authoring reference", editing.ParagraphOptions{Style: "Title", Text: editing.TextStyle{Color: "000000"}}); err != nil {
		return err
	}
	if _, err := doc.Normal("This document demonstrates reusable authoring operations with synthetic content. It covers literal values, navigation, tables, images, HTML and template composition."); err != nil {
		return err
	}
	if _, err := doc.Contents(2); err != nil {
		return err
	}
	if _, err := doc.Normal("Update the contents field in Word to calculate page numbers for the installed fonts and reader."); err != nil {
		return err
	}
	heading, err := doc.Heading("Exact values and table controls", 2, false)
	if err != nil {
		return err
	}
	if err := heading.Bookmark("measurements"); err != nil {
		return err
	}
	if _, err := doc.Caption("Table", "Synthetic measurements"); err != nil {
		return err
	}
	values := tabledata.Data{Headers: []string{"Identity", "Exact value"}, Rows: [][]string{{"A_LONG_IDENTIFIER_WITH_PARTITION_SUFFIX", "9007199254740993"}, {"Small measurement", "42"}, {"Unavailable measurement", ""}}}
	values, err = values.Sort(tabledata.SortKey{Column: 1, Kind: tabledata.Number, Descending: true})
	if err != nil {
		return err
	}
	table, err := doc.Table(values, editing.TableOptions{Columns: []editing.Column{{Width: "45%", SingleLine: true, Ellipsis: 28}, {Width: "55%"}}})
	if err != nil {
		return err
	}
	if err := table.CellStyle(1, 1, editing.TextStyle{}, "FFF0D5"); err != nil {
		return err
	}
	if err := table.Height(1, 280, false); err != nil {
		return err
	}
	if _, err := doc.Normal("The large integer remains literal and exact. The shortened identifier retains its full value in a Word ScreenTip. The empty value remains empty and does not become zero; the caller decides what it means."); err != nil {
		return err
	}
	if _, err := doc.Heading("Literal code and linked actions", 2, false); err != nil {
		return err
	}
	if _, err := doc.Code("SELECT identity, amount\nFROM example_measurements\nORDER BY identity;"); err != nil {
		return err
	}
	if _, err := actions.Add(doc, []actions.Entry{{Section: "Measurements", Text: "Review the synthetic difference", Risk: "Caller supplied", Impact: "Caller supplied", Priority: "Caller supplied", Bookmark: "measurements", Color: "FFF0D5"}}, actions.Labels{Section: "Section", Action: "Action", Risk: "Risk", Impact: "Impact", Priority: "Priority"}, editing.TableOptions{}); err != nil {
		return err
	}
	if _, err := doc.Normal("Action labels, ranking, visibility and acceptance come from the caller. Colors describe presentation and do not evaluate evidence."); err != nil {
		return err
	}
	if err := htmlcontent.Append(ctx, doc, strings.NewReader("<h2>Imported HTML content</h2><p>Prose can retain <b>emphasis</b>, <i>italic text</i> and a <a href='https://example.com/reference'>reference link</a>.</p><ul><li>First caller-authored item</li><li>Second caller-authored item</li></ul>"), htmlcontent.Options{}); err != nil {
		return err
	}
	if _, err := doc.Caption("Figure", "Generated sample graphic"); err != nil {
		return err
	}
	paragraph, err := doc.Paragraph("", editing.ParagraphOptions{})
	if err != nil {
		return err
	}
	graphic, err := sampleGraphic()
	if err != nil {
		return err
	}
	if err := paragraph.Image(graphic, 2880, 0); err != nil {
		return err
	}
	if _, err := doc.Normal("The graphic is generated in memory. Document assets and fonts are supplied by the caller rather than discovered from an application."); err != nil {
		return err
	}
	var main bytes.Buffer
	if err := doc.Write(ctx, &main); err != nil {
		return err
	}
	closing, err := editing.New(options)
	if err != nil {
		return err
	}
	if _, err := closing.Heading("Closing page", 1, false); err != nil {
		return err
	}
	if _, err := closing.Normal("Prepared for {{ RECIPIENT }}"); err != nil {
		return err
	}
	if _, err := closing.Paragraph("{{MARK}}", editing.ParagraphOptions{}); err != nil {
		return err
	}
	if _, err := closing.Normal("This page comes from a separate caller-supplied template. The main document owns page geometry, headers, footers and core properties."); err != nil {
		return err
	}
	var ending bytes.Buffer
	if err := closing.Write(ctx, &ending); err != nil {
		return err
	}
	rendered, err := doctemplate.Render(ctx, ending.Bytes(), map[string]string{"RECIPIENT": "Example recipient"}, doctemplate.Options{Document: options, Images: map[string]doctemplate.Image{"MARK": {Data: graphic, Width: 1440}}})
	if err != nil {
		return err
	}
	result, err := doctemplate.Compose(ctx, main.Bytes(), rendered)
	if err != nil {
		return err
	}
	receipt, err := publication.Save(ctx, path, false, publication.FromBytes(result))
	if err != nil {
		return err
	}
	fmt.Printf("Saved %s (%d bytes, SHA-512 %s)\n", path, receipt.Size, receipt.SHA512)
	return nil
}
func sampleGraphic() ([]byte, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, 240, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 240; x++ {
			fill := color.RGBA{R: 233, G: 229, B: 223, A: 255}
			if x >= 120 {
				fill = color.RGBA{R: 243, G: 190, B: 85, A: 255}
			}
			canvas.SetRGBA(x, y, fill)
		}
	}
	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
