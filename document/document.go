// Package document builds Word documents from literal text and explicit layout.
// Documents own their inputs and are intended for serial construction and writing.
package document

import (
	"fmt"
	"slices"
)

// Options defines page geometry in twips (1/1440 inch) and font sizes in half points.
// Fonts are referenced, never embedded or installed. Header and Footer are caller text.
// FallbackFont is an optional reader substitution hint, not a rendering guarantee.
type Options struct {
	PageWidth, PageHeight, Margin                    int
	Font, HeadingFont, SubheadingFont                string
	FallbackFont                                     string
	BodySize, HeadingSize, SubheadingSize, TableSize int
	Header, Footer, PageLabel                        string
}

// Tone is a presentation hint. A tone never evaluates evidence or changes text.
type Tone string

// Plain, Difference and Unavailable select restrained fills with black text.
const (
	Plain       Tone = ""
	Difference  Tone = "difference"
	Unavailable Tone = "unavailable"
)

// Cell contains exact text and an optional visual hint; numbers remain exact text.
type Cell struct {
	Text string
	Tone Tone
	// TextColor is an optional six-digit RGB color without a leading #.
	// Empty inherits the paragraph style; it does not change the fill tone.
	TextColor string
	// Bold adds direct emphasis; false preserves the paragraph style's emphasis.
	Bold bool
}

// Table contains a repeating header and explicit column widths in twips.
// Widths must be positive, match Headers, and fit the page's available width.
// Tables may span pages. Rows stay together when they fit on a page; a row taller
// than a page can still split. The renderer never silently drops values.
type Table struct {
	Headers []string
	Widths  []int
	Rows    [][]Cell
}

type block struct {
	style, text string
	table       *Table
	pageBreak   bool
}

// Document owns an ordered snapshot of report blocks. It is not concurrency safe.
// Write validates all input before sending bytes to its writer.
type Document struct {
	options Options
	blocks  []block
}

// New constructs a document with explicit options. Invalid dimensions or font sizes
// are rejected; report presets live in the separate report package.
func New(options Options) (*Document, error) {
	if err := validateOptions(options); err != nil {
		return nil, err
	}
	return &Document{options: options}, nil
}

func validateOptions(options Options) error {
	if options.PageWidth <= 0 || options.PageHeight <= 0 || options.Margin < 0 || options.Margin >= options.PageWidth/2 || options.Margin >= options.PageHeight/2 || options.PageWidth > 31680 || options.PageHeight > 31680 {
		return fmt.Errorf("invalid page dimensions or margins")
	}
	for _, size := range []int{options.BodySize, options.HeadingSize, options.SubheadingSize, options.TableSize} {
		if size < 12 || size > 144 {
			return fmt.Errorf("font sizes must be 12 to 144 half points")
		}
	}
	if options.Font == "" || options.HeadingFont == "" || options.SubheadingFont == "" {
		return fmt.Errorf("font names are required")
	}
	for _, font := range []string{options.Font, options.HeadingFont, options.SubheadingFont, options.FallbackFont} {
		if len(font) > 255 {
			return fmt.Errorf("font name exceeds 255 bytes")
		}
	}
	for _, text := range []string{options.Header, options.Footer, options.PageLabel} {
		if len(text) > 8192 {
			return fmt.Errorf("header, footer or page label exceeds 8 KiB")
		}
	}
	return nil
}

// Title appends a Word Title paragraph suitable for a report cover.
func (d *Document) Title(text string) { d.blocks = append(d.blocks, block{style: "Title", text: text}) }

// Subtitle appends secondary cover text in black using the Subtitle style.
func (d *Document) Subtitle(text string) {
	d.blocks = append(d.blocks, block{style: "Subtitle", text: text})
}

// Heading appends a navigable heading at level 1, 2 or 3. Write rejects other levels.
func (d *Document) Heading(text string, level int) {
	d.blocks = append(d.blocks, block{style: fmt.Sprintf("Heading%d", level), text: text})
}

// Paragraph appends literal prose. Newlines become soft breaks; tabs remain tabs.
// No text is interpreted as markup, formulas, fields or hyperlinks.
func (d *Document) Paragraph(text string) {
	d.blocks = append(d.blocks, block{style: "Normal", text: text})
}

// ParagraphWithNext appends literal prose with keep-next and keep-lines hints.
// It is useful before a table; the reader still controls final pagination and
// can split content taller than a page. Literal-text rules match Paragraph.
func (d *Document) ParagraphWithNext(text string) {
	d.blocks = append(d.blocks, block{style: "Lead", text: text})
}

// PageBreak starts the next block on a new page without creating a section.
func (d *Document) PageBreak() { d.blocks = append(d.blocks, block{pageBreak: true}) }

// AddTable takes a defensive copy so later caller edits cannot change the document.
// Invalid shape, widths, tones or text are reported by Write before publication.
func (d *Document) AddTable(table Table) {
	owned := table
	owned.Headers = slices.Clone(table.Headers)
	owned.Widths = slices.Clone(table.Widths)
	owned.Rows = make([][]Cell, len(table.Rows))
	for i, row := range table.Rows {
		owned.Rows[i] = slices.Clone(row)
	}
	d.blocks = append(d.blocks, block{table: &owned})
}
