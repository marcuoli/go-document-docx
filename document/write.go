package document

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

const wordNS = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
const xmlStart = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`

// Write serializes a complete DOCX ZIP to writer without closing the caller's
// writer. It propagates cancellation and short writes. A writer failure may leave
// partial bytes; use publication.Publish when an existing file must be preserved.
// This creates documents; it does not load arbitrary templates or render pages.
func (d *Document) Write(ctx context.Context, writer io.Writer) error {
	if d == nil || writer == nil {
		return fmt.Errorf("document and writer are required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.validate(ctx); err != nil {
		return err
	}
	body, err := d.body(ctx)
	if err != nil {
		return err
	}
	parts := []struct{ name, text string }{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", rootRelationships},
		{"word/document.xml", body},
		{"word/styles.xml", d.styles()},
		{"word/settings.xml", settings},
		{"word/fontTable.xml", d.fonts()},
		{"word/header1.xml", d.header()},
		{"word/footer1.xml", d.footer()},
		{"word/_rels/document.xml.rels", documentRelationships},
	}
	z := zip.NewWriter(contextWriter{ctx: ctx, writer: writer})
	for _, part := range parts {
		if err := ctx.Err(); err != nil {
			return errors.Join(err, z.Close())
		}
		w, err := z.Create(part.name)
		if err == nil {
			_, err = io.WriteString(w, xmlStart+part.text)
		}
		if err != nil {
			return errors.Join(err, z.Close())
		}
	}
	return errors.Join(z.Close(), ctx.Err())
}

type contextWriter struct {
	ctx    context.Context
	writer io.Writer
}

func (w contextWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := w.writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}

func (d *Document) validate(ctx context.Context) error {
	if err := validateOptions(d.options); err != nil {
		return err
	}
	textBytes, cells := 0, 0
	check := func(text string) error {
		textBytes += len(text)
		if textBytes > 64<<20 {
			return fmt.Errorf("document text exceeds 64 MiB")
		}
		if !utf8.ValidString(text) {
			return fmt.Errorf("document text is not UTF-8")
		}
		for _, r := range text {
			if !(r == 9 || r == 10 || r == 13 || r >= 32 && r <= 0xD7FF || r >= 0xE000 && r <= 0xFFFD || r >= 0x10000 && r <= 0x10FFFF) {
				return fmt.Errorf("document text contains an XML-invalid character")
			}
		}
		return nil
	}
	for _, text := range []string{d.options.Font, d.options.HeadingFont, d.options.SubheadingFont, d.options.FallbackFont, d.options.Header, d.options.Footer, d.options.PageLabel} {
		if err := check(text); err != nil {
			return err
		}
	}
	for _, b := range d.blocks {
		if err := ctx.Err(); err != nil {
			return err
		}
		if b.pageBreak {
			continue
		}
		if b.table == nil {
			switch b.style {
			case "Normal", "Title", "Subtitle", "Heading1", "Heading2", "Heading3":
			default:
				return fmt.Errorf("unsupported heading level")
			}
			if err := check(b.text); err != nil {
				return err
			}
			continue
		}
		t := b.table
		if len(t.Headers) == 0 || len(t.Headers) > 32 || len(t.Widths) != len(t.Headers) {
			return fmt.Errorf("table headers and widths must match")
		}
		width := 0
		for i, w := range t.Widths {
			if w <= 0 || w > d.options.PageWidth {
				return fmt.Errorf("invalid column width")
			}
			width += w
			if err := check(t.Headers[i]); err != nil {
				return err
			}
		}
		if width > d.options.PageWidth-2*d.options.Margin {
			return fmt.Errorf("table exceeds available page width")
		}
		cells += len(t.Headers)
		if cells > 1_000_000 {
			return fmt.Errorf("document exceeds one million cells")
		}
		for _, row := range t.Rows {
			if err := ctx.Err(); err != nil {
				return err
			}
			if len(row) != len(t.Headers) {
				return fmt.Errorf("table row does not match headers")
			}
			cells += len(row)
			if cells > 1_000_000 {
				return fmt.Errorf("document exceeds one million cells")
			}
			for _, c := range row {
				if c.Tone != Plain && c.Tone != Difference && c.Tone != Unavailable {
					return fmt.Errorf("unsupported cell tone")
				}
				if err := check(c.Text); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func escape(text string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(text))
	return b.String()
}

// Soft breaks deliberately disable full-line expansion in settings.xml. Text
// remains exact, including whitespace, without being interpreted as OOXML.
func textRun(text string) string {
	var b strings.Builder
	_ = writeText(context.Background(), &b, text)
	return b.String()
}

// Check cancellation and expanded XML size in small UTF-8 chunks. Splitting a
// large newline-heavy value into slices would allocate before the limit applies.
func writeText(ctx context.Context, b *strings.Builder, text string) error {
	b.WriteString(`<w:t xml:space="preserve">`)
	for offset := 0; offset < len(text); {
		if err := ctx.Err(); err != nil {
			return err
		}
		if b.Len() > 128<<20 {
			return fmt.Errorf("document XML exceeds 128 MiB")
		}
		switch text[offset] {
		case '\r', '\n', '\t':
			current := text[offset]
			offset++
			if current == '\r' && offset < len(text) && text[offset] == '\n' {
				offset++
			}
			b.WriteString(`</w:t>`)
			if current == '\t' {
				b.WriteString(`<w:tab/>`)
			} else {
				b.WriteString(`<w:br/>`)
			}
			b.WriteString(`<w:t xml:space="preserve">`)
		default:
			end := min(offset+4096, len(text))
			for end < len(text) && !utf8.RuneStart(text[end]) {
				end--
			}
			if i := strings.IndexAny(text[offset:end], "\r\n\t"); i >= 0 {
				end = offset + i
			}
			if err := xml.EscapeText(b, []byte(text[offset:end])); err != nil {
				return err
			}
			offset = end
		}
	}
	b.WriteString(`</w:t>`)
	if b.Len() > 128<<20 {
		return fmt.Errorf("document XML exceeds 128 MiB")
	}
	return nil
}

func writeParagraph(ctx context.Context, b *strings.Builder, style, text string) error {
	b.WriteString(`<w:p><w:pPr><w:pStyle w:val="` + style + `"/></w:pPr><w:r>`)
	if err := writeText(ctx, b, text); err != nil {
		return err
	}
	b.WriteString(`</w:r></w:p>`)
	return nil
}

func paragraph(style, text string) string {
	var b strings.Builder
	_ = writeParagraph(context.Background(), &b, style, text)
	return b.String()
}

func (d *Document) body(ctx context.Context) (string, error) {
	var b strings.Builder
	b.WriteString(`<w:document xmlns:w="` + wordNS + `" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body>`)
	for _, item := range d.blocks {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		switch {
		case item.pageBreak:
			b.WriteString(`<w:p><w:r><w:br w:type="page"/></w:r></w:p>`)
		case item.table != nil:
			if err := writeTable(ctx, &b, *item.table); err != nil {
				return "", err
			}
		default:
			if err := writeParagraph(ctx, &b, item.style, item.text); err != nil {
				return "", err
			}
		}
		if b.Len() > 128<<20 {
			return "", fmt.Errorf("document XML exceeds 128 MiB")
		}
	}
	o := d.options
	fmt.Fprintf(&b, `<w:sectPr><w:headerReference w:type="default" r:id="rIdHeader"/><w:footerReference w:type="default" r:id="rIdFooter"/><w:pgSz w:w="%d" w:h="%d"/><w:pgMar w:top="%d" w:right="%d" w:bottom="%d" w:left="%d" w:header="504" w:footer="576" w:gutter="0"/></w:sectPr></w:body></w:document>`, o.PageWidth, o.PageHeight, o.Margin, o.Margin, o.Margin, o.Margin)
	if b.Len() > 128<<20 {
		return "", fmt.Errorf("document XML exceeds 128 MiB")
	}
	return b.String(), nil
}

func writeTable(ctx context.Context, b *strings.Builder, t Table) error {
	width := 0
	for _, w := range t.Widths {
		width += w
	}
	fmt.Fprintf(b, `<w:tbl><w:tblPr><w:tblW w:w="%d" w:type="dxa"/><w:tblBorders><w:bottom w:val="single" w:sz="4" w:color="D6D2CE"/><w:insideH w:val="single" w:sz="4" w:color="E8E5E1"/></w:tblBorders><w:tblLayout w:type="fixed"/><w:tblCellMar><w:top w:w="70" w:type="dxa"/><w:left w:w="108" w:type="dxa"/><w:bottom w:w="70" w:type="dxa"/><w:right w:w="108" w:type="dxa"/></w:tblCellMar></w:tblPr><w:tblGrid>`, width)
	for _, w := range t.Widths {
		fmt.Fprintf(b, `<w:gridCol w:w="%d"/>`, w)
	}
	b.WriteString(`</w:tblGrid>`)
	row := func(cells []Cell, header bool, index int) error {
		b.WriteString(`<w:tr><w:trPr><w:cantSplit/>`)
		if header {
			b.WriteString(`<w:tblHeader/>`)
		}
		b.WriteString(`</w:trPr>`)
		for i, c := range cells {
			fill := "FFFFFF"
			if index%2 == 1 {
				fill = "F7F6F4"
			}
			if header {
				fill = "E9E5DF"
			}
			if c.Tone == Difference {
				fill = "FFF0D5"
			}
			if c.Tone == Unavailable {
				fill = "EDEBE8"
			}
			fmt.Fprintf(b, `<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/><w:shd w:val="clear" w:fill="%s"/><w:vAlign w:val="top"/></w:tcPr>`, t.Widths[i], fill)
			style := "TableText"
			if header {
				style = "TableHeader"
			}
			if err := writeParagraph(ctx, b, style, c.Text); err != nil {
				return err
			}
			b.WriteString(`</w:tc>`)
		}
		b.WriteString(`</w:tr>`)
		return nil
	}
	head := make([]Cell, len(t.Headers))
	for i, h := range t.Headers {
		head[i] = Cell{Text: h}
	}
	if err := row(head, true, 0); err != nil {
		return err
	}
	for i, r := range t.Rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := row(r, false, i); err != nil {
			return err
		}
		if b.Len() > 128<<20 {
			return fmt.Errorf("document XML exceeds 128 MiB")
		}
	}
	b.WriteString(`</w:tbl>` + paragraph("TableAfter", ""))
	return nil
}
