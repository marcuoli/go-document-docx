package editing

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mmonterroca/docxgo/v2/domain"
)

// TextStyle defines explicit run formatting. Size uses half-points; zero inherits.
// An empty Font or Color inherits. Nil effects inherit; non-nil effects explicitly enable/disable them.
type TextStyle struct {
	Font                            string
	Size                            int
	Color                           string
	Bold, Italic, Underline, Strike *bool
}

// ParagraphOptions controls new paragraphs. Nil optional values inherit; explicit
// zero alignment/spacing/indent values remain meaningful. All distances use twips.
type ParagraphOptions struct {
	Style         string
	Alignment     *domain.Alignment
	Indent        *domain.Indentation
	Before, After *int
	KeepNext      *bool
	Text          TextStyle
}

// Paragraph is an owned paragraph helper. Native exposes supported upstream edits.
// It is serial and belongs to the editor that created it.
type Paragraph struct {
	native                   domain.Paragraph
	owner                    *Document
	bookmark                 string
	links                    []link
	listKind                 string
	listLevel                int
	styles                   map[int]TextStyle
	alignment, before, after bool
}
type link struct {
	run                   int
	target, text, tooltip string
	internal              bool
}

// Native returns the owned upstream paragraph, not a copy.
func (p *Paragraph) Native() domain.Paragraph { return p.native }

// Paragraph appends literal text once. Translate before calling when needed.
func (d *Document) Paragraph(text string, options ParagraphOptions) (*Paragraph, error) {
	if err := validText(text); err != nil {
		return nil, err
	}
	if err := validateStyle(options.Text); err != nil {
		return nil, err
	}
	if options.Style != "" && !d.native.StyleManager().HasStyle(options.Style) {
		return nil, fmt.Errorf("unknown paragraph style %q", options.Style)
	}
	text = normalizedBreaks(text)
	p, err := d.native.AddParagraph()
	if err != nil {
		return nil, err
	}
	result := &Paragraph{native: p, owner: d, styles: map[int]TextStyle{0: options.Text}, alignment: options.Alignment != nil, before: options.Before != nil, after: options.After != nil}
	if options.Style != "" {
		err = p.SetStyle(options.Style)
	}
	if options.Alignment != nil {
		err = errors.Join(err, p.SetAlignment(*options.Alignment))
	}
	if options.Indent != nil {
		err = errors.Join(err, p.SetIndent(*options.Indent))
	}
	if options.Before != nil {
		err = errors.Join(err, p.SetSpacingBefore(*options.Before))
	}
	if options.After != nil {
		err = errors.Join(err, p.SetSpacingAfter(*options.After))
	}
	if options.KeepNext != nil {
		err = errors.Join(err, p.SetKeepNext(*options.KeepNext))
	}
	r, runErr := p.AddRun()
	if runErr != nil {
		return nil, errors.Join(err, runErr)
	}
	err = errors.Join(err, r.SetText(text), styleRun(r, options.Text))
	if err != nil {
		return nil, err
	}
	d.paragraphs = append(d.paragraphs, result)
	return result, nil
}

// Heading appends a black, navigable heading at level 1 through 9 and notifies
// HeadingProgress only after successful creation. Page breaks are explicit.
func (d *Document) Heading(text string, level int, pageBreak bool) (*Paragraph, error) {
	if level < 1 || level > 9 {
		return nil, fmt.Errorf("heading level must be 1 to 9")
	}
	if pageBreak {
		if err := d.PageBreak(); err != nil {
			return nil, err
		}
	}
	size, font := d.options.BodySize, d.options.SubheadingFont
	if level == 1 {
		size, font = d.options.HeadingSize, d.options.HeadingFont
	} else if level == 2 {
		size = d.options.SubheadingSize
	}
	// A template need not define every heading level. Direct formatting still
	// supplies the outline level during package amendment.
	keep := true
	left := domain.AlignmentLeft
	opts := ParagraphOptions{Alignment: &left, KeepNext: &keep, Text: TextStyle{Font: font, Size: size, Color: "000000", Bold: new(true)}}
	style := fmt.Sprintf("Heading%d", level)
	if d.native.StyleManager().HasStyle(style) {
		opts.Style = style
	}
	p, err := d.Paragraph(text, opts)
	if err != nil {
		return nil, err
	}
	p.listKind = "heading"
	p.listLevel = level - 1
	if d.HeadingProgress != nil {
		d.HeadingProgress(level, text)
	}
	return p, nil
}

// Code appends literal left-aligned monospace text. It never executes or translates it.
func (d *Document) Code(text string) (*Paragraph, error) {
	left := domain.AlignmentLeft
	return d.Paragraph(text, ParagraphOptions{Alignment: &left, Text: TextStyle{Font: "Consolas", Size: d.options.BodySize, Color: "000000"}})
}

// List appends one native list item at level 0 through 8. Numbered sequences share
// this document's numbering instance; bullets use a separate instance.
func (d *Document) List(text string, numbered bool, level int) (*Paragraph, error) {
	if level < 0 || level > 8 {
		return nil, fmt.Errorf("list level must be 0 to 8")
	}
	p, err := d.Paragraph(text, ParagraphOptions{})
	if err != nil {
		return nil, err
	}
	p.listKind = "bullet"
	if numbered {
		p.listKind = "decimal"
	}
	p.listLevel = level
	return p, nil
}

// Caption creates a per-document sequence and bookmark for a caller-supplied
// object label. Counters never share state across editors; the first number is one.
func (d *Document) Caption(label, text string) (*Paragraph, error) {
	if strings.TrimSpace(label) == "" {
		return nil, fmt.Errorf("caption label is required")
	}
	n := d.captionCounters[label] + 1
	p, err := d.Paragraph(fmt.Sprintf("%s %d: %s", label, n, text), ParagraphOptions{Text: TextStyle{Size: d.options.TableSize, Bold: new(true)}})
	if err != nil {
		return nil, err
	}
	d.captionCounters[label] = n
	if err := p.Bookmark(fmt.Sprintf("object_%d", len(d.paragraphs))); err != nil {
		return nil, err
	}
	return p, nil
}

// Bookmark marks the paragraph with a unique Word-compatible name. Duplicate
// names, including names from a loaded template, are rejected at Write.
func (p *Paragraph) Bookmark(name string) error {
	if len(name) == 0 || len(name) > 40 {
		return fmt.Errorf("bookmark length must be 1 to 40")
	}
	for i, r := range name {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9') {
			return fmt.Errorf("invalid bookmark name")
		}
	}
	p.bookmark = name
	return nil
}

// Link appends one styled literal link. An internal target names a bookmark;
// external targets are relationships and are never fetched. Tooltip text stays literal.
func (p *Paragraph) Link(text, target, tooltip string, internal bool) error {
	for _, s := range []string{text, target, tooltip} {
		if err := validText(s); err != nil {
			return err
		}
	}
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("link target is required")
	}
	if !internal && !strings.HasPrefix(target, "https://") && !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "mailto:") {
		return fmt.Errorf("unsupported external link scheme")
	}
	text = normalizedBreaks(text)
	index := len(p.native.Runs())
	r, err := p.native.AddRun()
	if err != nil {
		return err
	}
	if err := errors.Join(r.SetText(text), styleRun(r, TextStyle{Color: "315357", Underline: new(true)})); err != nil {
		return err
	}
	p.styles[index] = TextStyle{Color: "315357", Underline: new(true)}
	p.links = append(p.links, link{run: index, target: target, text: text, tooltip: tooltip, internal: internal})
	return nil
}

// Rule adds a restrained top border; Box adds all four borders and a neutral fill.
func (p *Paragraph) Rule(box bool) error {
	border := domain.BorderStyle{Style: domain.BorderSingle, Width: 6, Color: domain.ColorBlack}
	borders := domain.ParagraphBorders{Top: border}
	if box {
		borders.Left = border
		borders.Right = border
		borders.Bottom = border
	}
	err := p.native.SetBorders(borders)
	if box {
		err = errors.Join(err, p.native.SetShading(domain.Color{R: 242, G: 242, B: 242}))
	}
	return err
}

func validateStyle(s TextStyle) error {
	if s.Size != 0 && (s.Size < 12 || s.Size > 144) {
		return fmt.Errorf("font size must be 12 to 144 half-points")
	}
	if len(s.Font) > 255 {
		return fmt.Errorf("font name exceeds 255 bytes")
	}
	if err := validText(s.Font); err != nil {
		return err
	}
	if s.Color != "" {
		_, err := color(s.Color)
		return err
	}
	return nil
}
func color(hex string) (domain.Color, error) {
	if len(hex) != 6 {
		return domain.Color{}, fmt.Errorf("color must have six hex digits")
	}
	n, err := strconv.ParseUint(hex, 16, 24)
	if err != nil {
		return domain.Color{}, fmt.Errorf("invalid color")
	}
	return domain.Color{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n)}, nil
}
func styleRun(r domain.Run, s TextStyle) error {
	if err := validateStyle(s); err != nil {
		return err
	}
	var err error
	if s.Font != "" {
		err = r.SetFont(domain.Font{Name: s.Font})
	}
	if s.Size != 0 {
		err = errors.Join(err, r.SetSize(s.Size))
	}
	if s.Color != "" {
		c, e := color(s.Color)
		err = errors.Join(err, e, r.SetColor(c))
	}
	if s.Bold != nil {
		err = errors.Join(err, r.SetBold(*s.Bold))
	}
	if s.Italic != nil {
		err = errors.Join(err, r.SetItalic(*s.Italic))
	}
	if s.Strike != nil {
		err = errors.Join(err, r.SetStrike(*s.Strike))
	}
	if s.Underline != nil {
		underline := domain.UnderlineNone
		if *s.Underline {
			underline = domain.UnderlineSingle
		}
		err = errors.Join(err, r.SetUnderline(underline))
	}
	return err
}

func normalizedBreaks(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
}
