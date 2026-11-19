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
	// TableStyle optionally overrides literal table appearance; nil preserves legacy output.
	TableStyle                                       *TableStyle
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
	// Bookmark names this cell's literal paragraph as an internal destination.
	// Empty adds none. Write validates the shared Word name rules and uniqueness
	// across named cells and BookmarkNext requests before sending output.
	Bookmark string
	// LinkTarget names an exact bookmark in this document; forward references are
	// allowed. Nonempty applies the shared navigation color and underline, keeping
	// explicit Cell/CellRun colors, emphasis, size and literal text. ScreenTip then
	// annotates this same link. Write rejects invalid or unresolved targets.
	LinkTarget string
	Text       string
	Tone       Tone
	// TextColor is an optional six-digit RGB color without a leading #.
	// Empty inherits the paragraph style; it does not change the fill tone.
	TextColor string
	// Bold adds direct emphasis; false preserves the paragraph style's emphasis.
	Bold bool
	// Runs optionally supplies ordered literal spans within one cell paragraph.
	// Empty/nil Runs uses Text as before. Nonempty Runs requires empty Text;
	// Write rejects ambiguous input before output. AddTable deep-copies Runs.
	// Cell contains a slice: use keyed literals and compare fields as appropriate.
	Runs []CellRun
	// Alignment optionally overrides this cell paragraph with "left", "center",
	// "right" or "justify", for either Text or Runs. Empty preserves the style's
	// alignment and existing output. Write rejects other values before output.
	// Alignment never parses or reformats text; AddTable copies the value.
	Alignment string
	// ScreenTip is optional literal hover text for the whole cell paragraph.
	// With LinkTarget it annotates that styled navigation link; otherwise it
	// uses an internal self-anchor hyperlink without added link styling.
	// Empty preserves existing output. AddTable copies it;
	// Write validates UTF-8/XML and counts it in the document text/XML limits.
	// Display and interaction depend on the reader; no visible text is added.
	ScreenTip string
}

// Table contains a repeating header and explicit column widths in twips.
// Widths must be positive, match Headers, and fit the page's available width.
// Tables may span pages. Rows stay together when they fit on a page; a row taller
// than a page can still split. The renderer never silently drops values.
type Table struct {
	// FontSize optionally overrides header and body runs in half-points (12..144).
	// Zero inherits Options.TableSize through the paragraph styles. It changes
	// only this table's typography; text, colors and alignment remain literal.
	FontSize int
	Headers  []string
	Widths   []int
	Rows     [][]Cell
}

type block struct {
	style, text     string
	bookmarks       []string
	table           *Table
	pageBreak       bool
	pageBreakBefore bool
	codeTable       bool
}

// Document owns an ordered snapshot of report blocks. It is not concurrency safe.
// Write validates all input before sending bytes to its writer.
type Document struct {
	options          Options
	blocks           []block
	pendingBookmarks []string
}

// New constructs a document with explicit options. Invalid dimensions or font sizes
// are rejected; report presets live in the separate report package.
func New(options Options) (*Document, error) {
	if err := validateOptions(options); err != nil {
		return nil, err
	}
	if options.TableStyle != nil {
		owned := *options.TableStyle
		options.TableStyle = &owned
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
	if options.TableStyle != nil {
		return validateTableStyle(*options.TableStyle)
	}
	return nil
}

// Title appends a Word Title paragraph suitable for a report cover.
func (d *Document) Title(text string) { d.appendParagraph(block{style: "Title", text: text}) }

// Subtitle appends secondary cover text in black using the Subtitle style.
func (d *Document) Subtitle(text string) {
	d.appendParagraph(block{style: "Subtitle", text: text})
}

// Heading appends a navigable heading at level 1, 2 or 3. Write rejects other levels.
func (d *Document) Heading(text string, level int) {
	d.appendParagraph(block{style: fmt.Sprintf("Heading%d", level), text: text})
}

// HeadingOnNewPage appends a navigable level 1, 2 or 3 heading with a page
// boundary on that paragraph. It avoids a break-only paragraph spilling after a
// full table; the preceding table's spacer is omitted. Write rejects other levels.
// Literal-text and keep-next behavior match Heading; no new section is created.
func (d *Document) HeadingOnNewPage(text string, level int) {
	d.appendParagraph(block{style: fmt.Sprintf("Heading%d", level), text: text, pageBreakBefore: true})
}

// Paragraph appends literal prose. Newlines become soft breaks; tabs remain tabs.
// No text is interpreted as markup, formulas, fields or hyperlinks.
func (d *Document) Paragraph(text string) {
	d.appendParagraph(block{style: "Normal", text: text})
}

// ParagraphWithNext appends literal prose with keep-next and keep-lines hints.
// It is useful before a table; the reader still controls final pagination and
// can split content taller than a page. Literal-text rules match Paragraph.
func (d *Document) ParagraphWithNext(text string) {
	d.appendParagraph(block{style: "Lead", text: text})
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
		for j := range owned.Rows[i] {
			owned.Rows[i][j].Runs = slices.Clone(row[j].Runs)
		}
	}
	d.blocks = append(d.blocks, block{table: &owned})
}
