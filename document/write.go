package document

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/marcuoli/go-document-docx/internal/navigation"
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
	names, err := d.navigationNames(ctx)
	if err != nil {
		return err
	}
	body, err := d.body(ctx, names)
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
	for _, text := range []string{d.options.Font, d.options.HeadingFont, d.options.SubheadingFont, d.options.FallbackFont, d.options.Header, d.options.Footer, d.options.PageLabel, tableStyleFor(d.options).HeaderFont} {
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
		for _, name := range b.bookmarks {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := check(name); err != nil {
				return err
			}
		}
		if b.table == nil {
			switch b.style {
			case "Normal", "Lead", "Title", "Subtitle", "Heading1", "Heading2", "Heading3":
			default:
				return fmt.Errorf("unsupported heading level")
			}
			if err := check(b.text); err != nil {
				return err
			}
			for _, run := range b.runs {
				if err := ctx.Err(); err != nil {
					return err
				}
				if !validTextColor(run.TextColor) {
					return fmt.Errorf("paragraph run text color must contain six hexadecimal RGB digits")
				}
				if err := check(run.Text); err != nil {
					return err
				}
			}
			continue
		}
		t := b.table
		if t.FontSize != 0 && (t.FontSize < 12 || t.FontSize > 144) {
			return fmt.Errorf("table font size must be zero or 12 to 144 half points")
		}
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
				if err := validateCellTextColor(c.TextColor); err != nil {
					return err
				}
				switch c.Alignment {
				case "", "left", "center", "right", "justify":
				default:
					return fmt.Errorf("invalid cell paragraph alignment")
				}
				for _, name := range []string{c.Bookmark, c.LinkTarget} {
					if err := check(name); err != nil {
						return err
					}
				}
				if err := check(c.ScreenTip); err != nil {
					return err
				}
				if err := check(c.Text); err != nil {
					return err
				}
				if len(c.Runs) > 0 && c.Text != "" {
					return fmt.Errorf("cell Text and nonempty Runs are mutually exclusive")
				}
				for _, run := range c.Runs {
					if err := ctx.Err(); err != nil {
						return err
					}
					if err := validateCellTextColor(run.TextColor); err != nil {
						return err
					}
					if err := check(run.Text); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func validateCellTextColor(color string) error {
	if !validTextColor(color) {
		return fmt.Errorf("cell text color must contain six hexadecimal RGB digits")
	}
	return nil
}

func validTextColor(color string) bool {
	if color == "" {
		return true
	}
	if len(color) != 6 {
		return false
	}
	_, err := hex.DecodeString(color)
	return err == nil
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

func writeParagraph(ctx context.Context, b *strings.Builder, item block, color string, bold bool, firstBookmarkID int) error {
	b.WriteString(`<w:p><w:pPr><w:pStyle w:val="` + item.style + `"/>`)
	if item.pageBreakBefore {
		b.WriteString(`<w:pageBreakBefore/>`)
	}
	b.WriteString(`</w:pPr>`)
	for i, name := range item.bookmarks {
		if err := writeBookmarkStart(ctx, b, name, firstBookmarkID+i); err != nil {
			return err
		}
	}
	if len(item.runs) == 0 {
		if err := writeRun(ctx, b, item.text, color, bold, false, 0, false); err != nil {
			return err
		}
	} else {
		previousCR := false
		for _, run := range item.runs {
			text := continuedRunText(run.Text, &previousCR)
			if err := writeRun(ctx, b, text, run.TextColor, false, false, 0, false); err != nil {
				return err
			}
		}
	}
	for i := len(item.bookmarks) - 1; i >= 0; i-- {
		if err := writeBookmarkEnd(ctx, b, firstBookmarkID+i); err != nil {
			return err
		}
	}
	b.WriteString(`</w:p>`)
	return nil
}

func writeRun(ctx context.Context, b *strings.Builder, text, color string, bold, italic bool, size int, underline bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.WriteString(`<w:r>`)
	if bold || italic || color != "" || size != 0 || underline {
		// Emphasis precedes color in the OOXML run-property sequence. Empty
		// formatting keeps inherited styles and the established literal path.
		b.WriteString(`<w:rPr>`)
		if bold {
			b.WriteString(`<w:b/>`)
		}
		if italic {
			b.WriteString(`<w:i/>`)
		}
		if color != "" {
			b.WriteString(`<w:color w:val="` + color + `"/>`)
		}
		if size != 0 {
			// Explicit run sizes survive template style imports. Keep complex-script
			// text at the same size without changing the document-wide styles.
			fmt.Fprintf(b, "<w:sz w:val=\"%d\"/><w:szCs w:val=\"%d\"/>", size, size)
		}
		if underline {
			b.WriteString(`<w:u w:val="single"/>`)
		}
		b.WriteString(`</w:rPr>`)
	}
	if err := writeText(ctx, b, text); err != nil {
		return err
	}
	b.WriteString(`</w:r>`)
	return nil
}

func writeCellParagraph(ctx context.Context, b *strings.Builder, style string, cell Cell, color string, size, bookmarkID int) error {
	if len(cell.Runs) == 0 && cell.Alignment == "" && size == 0 && cell.ScreenTip == "" && cell.Bookmark == "" && cell.LinkTarget == "" {
		return writeParagraph(ctx, b, block{style: style, text: cell.Text}, color, cell.Bold, 0)
	}
	b.WriteString(`<w:p><w:pPr><w:pStyle w:val="` + style + `"/>`)
	if cell.Alignment != "" {
		// Alignment is an allowlisted literal paragraph override, independent of
		// content and run formatting. Empty keeps inherited styles byte-for-byte.
		alignment := cell.Alignment
		if alignment == "justify" {
			alignment = "both" // OOXML names justified paragraph alignment "both".
		}
		b.WriteString(`<w:jc w:val="` + alignment + `"/>`)
	}
	b.WriteString(`</w:pPr>`)
	bookmark := cell.Bookmark
	if bookmark == "" && cell.ScreenTip != "" && cell.LinkTarget == "" {
		bookmark = fmt.Sprintf("DocxCellTip%d", bookmarkID)
	}
	if bookmark != "" {
		if err := writeBookmarkStart(ctx, b, bookmark, bookmarkID); err != nil {
			return err
		}
	}
	target := cell.LinkTarget
	if target == "" && cell.ScreenTip != "" {
		target = bookmark
	}
	if target != "" {
		fmt.Fprintf(b, `<w:hyperlink w:anchor="%s"`, target)
		if cell.ScreenTip != "" {
			b.WriteString(` w:tooltip="`)
			if err := writeScreenTipAttribute(ctx, b, cell.ScreenTip); err != nil {
				return err
			}
			b.WriteString(`"`)
		}
		b.WriteString(` w:history="0">`)
	}

	runs := cell.Runs
	if len(runs) == 0 {
		runs = []CellRun{{Text: cell.Text}}
	}
	previousCR := false
	for _, run := range runs {
		// Run colors override presentation only. In particular, a literal label
		// cannot imply availability or change another run's inherited color.
		runColor := run.TextColor
		if runColor == "" {
			runColor = color
		}
		if cell.LinkTarget != "" && runColor == "" {
			runColor = navigation.LinkColor
		}
		text := continuedRunText(run.Text, &previousCR)
		if err := writeRun(ctx, b, text, runColor, cell.Bold, run.Italic, size, cell.LinkTarget != ""); err != nil {
			return err
		}
	}
	if target != "" {
		b.WriteString(`</w:hyperlink>`)
	}
	if bookmark != "" {
		if err := writeBookmarkEnd(ctx, b, bookmarkID); err != nil {
			return err
		}
	}
	b.WriteString(`</w:p>`)
	return nil
}

// continuedRunText keeps a split CRLF as one soft break. Empty runs preserve
// the pending CR until the next nonempty run arrives.
func continuedRunText(text string, previousCR *bool) string {
	if text == "" {
		return text
	}
	skipLF := *previousCR && text[0] == '\n'
	*previousCR = text[len(text)-1] == '\r'
	if skipLF {
		return text[1:]
	}
	return text
}

func paragraph(style, text string) string {
	var b strings.Builder
	_ = writeParagraph(context.Background(), &b, block{style: style, text: text}, "", false, 0)
	return b.String()
}

func (d *Document) body(ctx context.Context, names map[string]string) (string, error) {
	var b strings.Builder
	bookmarks := bookmarkState{reserved: names}
	b.WriteString(`<w:document xmlns:w="` + wordNS + `" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body>`)
	for i, item := range d.blocks {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		switch {
		case item.pageBreak:
			b.WriteString(`<w:p><w:r><w:br w:type="page"/></w:r></w:p>`)
		case item.table != nil:
			spaceAfter := i+1 >= len(d.blocks) || !d.blocks[i+1].pageBreakBefore
			if err := writeTable(ctx, &b, *item.table, d.options, spaceAfter, item.codeTable, &bookmarks); err != nil {
				return "", err
			}
		default:
			// Paragraph names already belong to the document. Reserve consecutive
			// IDs without allocating another record slice for a large bookmark queue.
			firstID := bookmarks.id + 1
			bookmarks.id += len(item.bookmarks)
			if err := writeParagraph(ctx, &b, item, "", false, firstID); err != nil {
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

func writeTable(ctx context.Context, b *strings.Builder, t Table, options Options, spaceAfter, code bool, bookmarks *bookmarkState) error {
	appearance := tableStyleFor(options)
	if code {
		// The code palette is local to this block; ordinary table defaults and
		// caller appearance values must not be changed by a code table.
		appearance.HeaderFill = "FFFFFF"
		appearance.BodyFill = "000000"
		appearance.AlternateFill = "000000"
		appearance.HeaderTextColor = "000000"
		appearance.VerticalAlignment = "top"
	}
	width := 0
	for _, w := range t.Widths {
		width += w
	}
	fmt.Fprintf(b, `<w:tbl><w:tblPr><w:tblW w:w="%d" w:type="dxa"/><w:tblBorders>`, width)
	borders := []string{"bottom", "insideH"}
	if appearance.GridBorders {
		borders = []string{"top", "left", "bottom", "right", "insideH", "insideV"}
	}
	for _, edge := range borders {
		color := appearance.BorderColor
		if options.TableStyle == nil && edge == "insideH" {
			color = "E8E5E1"
		}
		fmt.Fprintf(b, `<w:%s w:val="single" w:sz="4" w:color="%s"/>`, edge, color)
	}
	fmt.Fprintf(b, `</w:tblBorders><w:tblLayout w:type="fixed"/><w:tblCellMar><w:top w:w="%d" w:type="dxa"/><w:left w:w="%d" w:type="dxa"/><w:bottom w:w="%d" w:type="dxa"/><w:right w:w="%d" w:type="dxa"/></w:tblCellMar></w:tblPr><w:tblGrid>`, appearance.PaddingTop, appearance.PaddingLeft, appearance.PaddingBottom, appearance.PaddingRight)
	for _, w := range t.Widths {
		fmt.Fprintf(b, `<w:gridCol w:w="%d"/>`, w)
	}
	b.WriteString(`</w:tblGrid>`)
	row := func(cells []Cell, header bool, index int) error {
		b.WriteString(`<w:tr><w:trPr>`)
		if code && !header {
			// Unlike a short data row, source code can be many pages long.
			// Explicitly allow splitting instead of keeping the whole cell together.
			b.WriteString(`<w:cantSplit w:val="0"/>`)
		} else {
			b.WriteString(`<w:cantSplit/>`)
		}
		if header {
			b.WriteString(`<w:tblHeader/>`)
		}
		b.WriteString(`</w:trPr>`)
		for i, c := range cells {
			fill := appearance.BodyFill
			if index%2 == 1 {
				fill = appearance.AlternateFill
			}
			if header {
				fill = appearance.HeaderFill
			}
			if c.Tone == Difference {
				fill = "FFF0D5"
			}
			if c.Tone == Unavailable {
				fill = "EDEBE8"
			}
			fmt.Fprintf(b, `<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/><w:shd w:val="clear" w:fill="%s"/><w:vAlign w:val="%s"/></w:tcPr>`, t.Widths[i], fill, appearance.VerticalAlignment)
			style := "TableText"
			if header {
				style = "TableHeader"
			}
			color := c.TextColor
			if code {
				style = "CodeText"
				if header {
					style = "CodeHeader"
				}
				color = "FFFFFF"
			}
			if header {
				color = appearance.HeaderTextColor
			}
			bookmarkID := 0
			if c.Bookmark != "" || c.ScreenTip != "" && c.LinkTarget == "" {
				var err error
				bookmarkID, err = bookmarks.next(ctx, c.Bookmark == "")
				if err != nil {
					return err
				}
			}
			if err := writeCellParagraph(ctx, b, style, c, color, t.FontSize, bookmarkID); err != nil {
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
	b.WriteString(`</w:tbl>`)
	// A trailing spacer can itself spill onto a new page. The following heading
	// owns its page boundary, so it must directly follow the table in this case.
	if spaceAfter {
		b.WriteString(paragraph("TableAfter", ""))
	}
	return nil
}
