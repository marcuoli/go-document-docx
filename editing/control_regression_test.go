package editing_test

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/internal/opc"
	"github.com/marcuoli/go-document-docx/report"
	"github.com/marcuoli/go-document-docx/tabledata"
	"github.com/mmonterroca/docxgo/v2/domain"
)

func TestExplicitOverridesGutterAndCellGeometry(t *testing.T) {
	base, _ := editing.New(report.Defaults())
	_, _ = base.Paragraph("Existing prose", editing.ParagraphOptions{})
	var b bytes.Buffer
	if err := base.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	pkg, _ := opc.Read(context.Background(), b.Bytes())
	styles, _ := pkg.XML("word/styles.xml")
	for _, style := range opc.Elements(styles.Root(), opc.Word, "style") {
		if opc.Attr(style, opc.Word, "styleId") == "Normal" {
			pr := opc.Properties(style, "pPr")
			opc.Ordered(pr, "jc", opc.ParagraphOrder).CreateAttr("w:val", "both")
			r := opc.Ordered(style, "rPr", opc.StyleOrder)
			opc.Ensure(r, "color").CreateAttr("w:val", "CC0000")
			opc.Ensure(r, "b").CreateAttr("w:val", "1")
		}
	}
	if err := pkg.SetXML("word/styles.xml", styles); err != nil {
		t.Fatal(err)
	}
	xml, _ := pkg.XML("word/document.xml")
	for _, m := range opc.Elements(xml.Root(), opc.Word, "pgMar") {
		m.CreateAttr("w:gutter", "900")
	}
	if err := pkg.SetXML("word/document.xml", xml); err != nil {
		t.Fatal(err)
	}
	b.Reset()
	if err := pkg.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	editor, err := editing.Open(context.Background(), b.Bytes(), report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	left := domain.AlignmentLeft
	zero := 0
	_, err = editor.Paragraph("Explicit overrides", editing.ParagraphOptions{Alignment: &left, Before: &zero, Text: editing.TextStyle{Color: "000000", Bold: new(false)}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = editor.Table(tabledata.Data{Headers: []string{"Key", "Value"}, Rows: [][]string{{"Exact", "9007199254740993"}}}, editing.TableOptions{Columns: []editing.Column{{Alignment: &left, SingleLine: true}, {}}})
	if err != nil {
		t.Fatal(err)
	}
	b.Reset()
	if err := editor.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	output, _ := opc.Read(context.Background(), b.Bytes())
	document, err := output.XML("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	p := opc.Elements(document.Root(), opc.Word, "p")[1]
	pr := opc.Child(p, opc.Word, "pPr")
	if opc.Attr(opc.Child(pr, opc.Word, "jc"), opc.Word, "val") != "left" {
		t.Fatal("explicit left ignored")
	}
	rpr := opc.Elements(p, opc.Word, "rPr")[0]
	if opc.Attr(opc.Child(rpr, opc.Word, "color"), opc.Word, "val") != "000000" || opc.Attr(opc.Child(rpr, opc.Word, "b"), opc.Word, "val") != "0" {
		t.Fatal("explicit black/false ignored")
	}
	for _, m := range opc.Elements(document.Root(), opc.Word, "pgMar") {
		if opc.Attr(m, opc.Word, "gutter") != "900" {
			t.Fatal("gutter lost")
		}
	}
	for _, cp := range opc.Elements(document.Root(), opc.Word, "tcPr") {
		if flag := opc.Child(cp, opc.Word, "noWrap"); flag != nil {
			if opc.Attr(flag, opc.Word, "val") != "1" || opc.Attr(opc.Child(cp, opc.Word, "tcFitText"), opc.Word, "val") != "0" {
				t.Fatal("single-line flag mismatch")
			}
		}
	}
	table := opc.Elements(document.Root(), opc.Word, "tbl")[0]
	grid := opc.Child(table, opc.Word, "tblGrid")
	total := 0
	for _, col := range grid.ChildElements() {
		n, err := strconv.Atoi(opc.Attr(col, opc.Word, "w"))
		if err != nil {
			t.Fatal(err)
		}
		total += n
	}
	if total != report.Defaults().PageWidth-2*report.Defaults().Margin-900 {
		t.Fatal("gutter absent from budget", total)
	}
	for _, row := range opc.Elements(table, opc.Word, "tr") {
		sum := 0
		for _, cell := range row.ChildElements() {
			if cell.Tag != "tc" {
				continue
			}
			width := opc.Child(opc.Child(cell, opc.Word, "tcPr"), opc.Word, "tcW")
			n, _ := strconv.Atoi(opc.Attr(width, opc.Word, "w"))
			sum += n
		}
		if sum != total {
			t.Fatal("row/grid geometry mismatch")
		}
	}
}
func TestHiddenInvalidFullCellTextRejected(t *testing.T) {
	for _, value := range []string{"AAAAA\x01BBBBB", string([]byte{'A', 'A', 0xff, 'B', 'B'})} {
		d, _ := editing.New(report.Defaults())
		if _, err := d.Table(tabledata.Data{Headers: []string{"Value"}, Rows: [][]string{{value}}}, editing.TableOptions{Columns: []editing.Column{{Ellipsis: 4}}}); err == nil {
			t.Fatal("invalid full tooltip accepted")
		}
	}
}
func TestNewStyleAndCRLFBreaks(t *testing.T) {
	d, _ := editing.New(report.Defaults())
	if _, err := d.ParagraphStyle("Small", "Small body", "Normal", "Arial", 16); err != nil {
		t.Fatal(err)
	}
	_, err := d.Paragraph("First\r\nSecond", editing.ParagraphOptions{Style: "Small"})
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := d.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	p, _ := opc.Read(context.Background(), b.Bytes())
	xml, _ := p.XML("word/document.xml")
	if len(opc.Elements(xml.Root(), opc.Word, "br")) != 1 {
		t.Fatal("CRLF expanded twice")
	}
	if !strings.Contains(string(p.Parts["word/styles.xml"]), "Small") {
		t.Fatal("custom style not saved")
	}
}

func TestDerivedStylePreservesExplicitBackendDefaults(t *testing.T) {
	d, _ := editing.New(report.Defaults())
	if _, err := d.ParagraphStyle("StandardSize", "Standard size", "Normal", "Calibri", 24); err != nil {
		t.Fatal(err)
	}
	_, _ = d.Paragraph("Styled text", editing.ParagraphOptions{Style: "StandardSize"})
	var b bytes.Buffer
	if err := d.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	p, _ := opc.Read(context.Background(), b.Bytes())
	styles, err := p.XML("word/styles.xml")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range opc.Elements(styles.Root(), opc.Word, "style") {
		if opc.Attr(style, opc.Word, "styleId") == "StandardSize" {
			pr := opc.Child(style, opc.Word, "rPr")
			if opc.Attr(opc.Child(pr, opc.Word, "rFonts"), opc.Word, "ascii") != "Calibri" || opc.Attr(opc.Child(pr, opc.Word, "sz"), opc.Word, "val") != "24" {
				t.Fatal("explicit style defaults lost")
			}
			return
		}
	}
	t.Fatal("style absent")
}

func TestExplicitCellAlignmentAppliesToEveryParagraph(t *testing.T) {
	d, _ := editing.New(report.Defaults())
	table, err := d.Table(tabledata.Data{Headers: []string{"Value"}, Rows: [][]string{{"First"}}}, editing.TableOptions{})
	if err != nil {
		t.Fatal(err)
	}
	row, _ := table.Native().Row(1)
	cell, _ := row.Cell(0)
	second, err := cell.AddParagraph()
	if err != nil {
		t.Fatal(err)
	}
	run, err := second.AddRun()
	if err != nil {
		t.Fatal(err)
	}
	if err := run.SetText("Second"); err != nil {
		t.Fatal(err)
	}
	for _, p := range cell.Paragraphs() {
		if err := p.SetAlignment(domain.AlignmentCenter); err != nil {
			t.Fatal(err)
		}
	}
	left := domain.AlignmentLeft
	if err := table.Align(1, 0, &left, nil); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := d.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	output, _ := opc.Read(context.Background(), b.Bytes())
	xml, _ := output.XML("word/document.xml")
	cells := opc.Elements(xml.Root(), opc.Word, "tc")
	paragraphs := opc.Elements(cells[1], opc.Word, "p")
	if len(paragraphs) != 2 {
		t.Fatal("paragraph count changed")
	}
	for _, p := range paragraphs {
		pr := opc.Child(p, opc.Word, "pPr")
		if pr == nil || opc.Child(pr, opc.Word, "jc") == nil || opc.Attr(opc.Child(pr, opc.Word, "jc"), opc.Word, "val") != "left" {
			t.Fatal("explicit left absent")
		}
	}
}
