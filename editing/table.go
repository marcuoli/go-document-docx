package editing

import (
	"errors"
	"fmt"

	"github.com/marcuoli/go-document-docx/tabledata"
	"github.com/mmonterroca/docxgo/v2/domain"
)

// Column controls an explicit table column. Width accepts tabledata.Widths units.
// Ellipsis is opt-in display shortening; the full value is retained as a ScreenTip.
type Column struct {
	Width      string
	Alignment  *domain.Alignment
	SingleLine bool
	Ellipsis   int
	Format     tabledata.Format
}

// Margins defines cell padding in twips. Zero is an explicit zero distance.
type Margins struct{ Top, Left, Bottom, Right int }

// TableOptions controls newly generated tables. Indent and CapWidth use twips;
// zero cap uses the established 17.19 cm cap. Nil margins use restrained defaults.
type TableOptions struct {
	Columns          []Column
	Indent, CapWidth int
	Margins          *Margins
}

// Table owns an upstream table and its explicit layout. Body data is defensively
// copied before rendering. Native changes share ownership and must remain serial.
type Table struct {
	native      domain.Table
	owner       *Document
	widths      []int
	options     TableOptions
	tips        map[[2]int]string
	displays    map[[2]int]string
	cellMargins map[[2]int]Margins
	styles      map[[2]int]TextStyle
	links       map[[2]int]link
	alignments  map[[2]int]cellAlignment
	heights     map[int]rowHeight
}

// Native exposes supported upstream table operations such as cell merges.
func (t *Table) Native() domain.Table { return t.native }

// Table appends a rectangular table with repeated headers, exact literal values,
// fixed geometry and restrained row bands. Empty body data produces just its header.
func (d *Document) Table(data tabledata.Data, options TableOptions) (*Table, error) {
	owned, err := data.Clone()
	if err != nil {
		return nil, err
	}
	n := len(owned.Headers)
	if len(options.Columns) == 0 {
		options.Columns = make([]Column, n)
	} else if len(options.Columns) != n {
		return nil, fmt.Errorf("column metadata must match table")
	}
	options.Columns = append([]Column(nil), options.Columns...)
	if options.Margins != nil {
		m := *options.Margins
		options.Margins = &m
		for _, v := range []int{m.Top, m.Left, m.Bottom, m.Right} {
			if v < 0 || v > 31680 {
				return nil, fmt.Errorf("invalid cell margins")
			}
		}
	}
	specs := make([]string, n)
	for i, c := range options.Columns {
		specs[i] = c.Width
		if c.Ellipsis != 0 && c.Ellipsis < 4 {
			return nil, fmt.Errorf("invalid ellipsis limit")
		}
	}
	sections := d.native.Sections()
	if len(sections) == 0 {
		return nil, fmt.Errorf("document has no section")
	}
	section := sections[len(sections)-1]
	page, m := section.PageSize(), section.Margins()
	capWidth := options.CapWidth
	if capWidth == 0 {
		capWidth = 9746
	}
	gutter := 0
	if index := len(sections) - 1; index < len(d.gutters) {
		gutter = d.gutters[index]
	}
	available, err := tabledata.AvailableWidth(page.Width, m.Left, m.Right, gutter, options.Indent, capWidth)
	if err != nil {
		return nil, err
	}
	widths, err := tabledata.Widths(available, specs)
	if err != nil {
		return nil, err
	}
	tips, displays := map[[2]int]string{}, map[[2]int]string{}
	for r, row := range owned.Rows {
		for c, text := range row {
			formatted, e := tabledata.Value(text, options.Columns[c].Format)
			if e != nil {
				return nil, fmt.Errorf("row %d column %d: %w", r, c, e)
			}
			if err := validText(formatted); err != nil {
				return nil, err
			}
			if limit := options.Columns[c].Ellipsis; limit > 0 {
				display, full, e := tabledata.Shorten(formatted, limit)
				if e != nil {
					return nil, e
				}
				if display != full {
					key := [2]int{r + 1, c}
					tips[key] = full
					displays[key] = display
				}
				formatted = display
			}
			if err := validText(formatted); err != nil {
				return nil, err
			}
			row[c] = formatted
		}
	}
	for _, h := range owned.Headers {
		if err := validText(h); err != nil {
			return nil, err
		}
	}
	native, err := d.native.AddTable(len(owned.Rows)+1, n)
	if err != nil {
		return nil, err
	}
	t := &Table{native: native, owner: d, widths: widths, options: options, tips: tips, displays: displays, styles: map[[2]int]TextStyle{}}
	if err := native.SetWidth(domain.TableWidth{Type: domain.WidthDXA, Value: available}); err != nil {
		return nil, err
	}
	matrix := append([][]string{owned.Headers}, owned.Rows...)
	for r, values := range matrix {
		row, err := native.Row(r)
		if err != nil {
			return nil, err
		}
		if err := errors.Join(row.SetCantSplit(true), row.SetRepeatAsHeader(r == 0)); err != nil {
			return nil, err
		}
		for c, text := range values {
			t.styles[[2]int{r, c}] = TextStyle{Font: d.options.Font, Size: d.options.TableSize, Bold: new(r == 0), Color: "000000"}
			cell, err := row.Cell(c)
			if err != nil {
				return nil, err
			}
			if err := cell.SetWidth(widths[c]); err != nil {
				return nil, err
			}
			paragraphs := cell.Paragraphs()
			var paragraph domain.Paragraph
			if len(paragraphs) == 0 {
				paragraph, err = cell.AddParagraph()
			} else {
				paragraph = paragraphs[0]
			}
			if err != nil {
				return nil, err
			}
			run, err := paragraph.AddRun()
			if err != nil {
				return nil, err
			}
			if err := errors.Join(run.SetText(text), styleRun(run, TextStyle{Font: d.options.Font, Size: d.options.TableSize, Bold: new(r == 0), Color: "000000"})); err != nil {
				return nil, err
			}
			if a := options.Columns[c].Alignment; a != nil {
				if err := paragraph.SetAlignment(*a); err != nil {
					return nil, err
				}
			}
			fill := domain.ColorWhite
			if r == 0 {
				fill = domain.Color{R: 233, G: 229, B: 223}
			} else if r%2 == 0 {
				fill = domain.Color{R: 247, G: 246, B: 244}
			}
			if err := cell.SetShading(fill); err != nil {
				return nil, err
			}
		}
	}
	d.tables = append(d.tables, t)
	return t, nil
}

// CellStyle changes run properties and optional fill for a zero-based row/column;
// row zero is the header. Values remain unchanged.
func (t *Table) CellStyle(row, column int, style TextStyle, fill string) error {
	if err := validateStyle(style); err != nil {
		return err
	}
	r, err := t.native.Row(row)
	if err != nil {
		return err
	}
	c, err := r.Cell(column)
	if err != nil {
		return err
	}
	if fill != "" {
		color, err := color(fill)
		if err != nil {
			return err
		}
		if err := c.SetShading(color); err != nil {
			return err
		}
	}
	for _, p := range c.Paragraphs() {
		for _, run := range p.Runs() {
			if err := styleRun(run, style); err != nil {
				return err
			}
		}
	}
	t.styles[[2]int{row, column}] = mergeStyle(t.styles[[2]int{row, column}], style)
	return nil
}

// RowStyle applies CellStyle to every cell in one row.
func (t *Table) RowStyle(row int, style TextStyle, fill string) error {
	r, err := t.native.Row(row)
	if err != nil {
		return err
	}
	for c := range r.Cells() {
		if err := t.CellStyle(row, c, style, fill); err != nil {
			return err
		}
	}
	return nil
}

// HeaderRows marks an explicit contiguous run of repeated header rows. Zero
// disables repetition. Word repeats row properties rather than table properties.
func (t *Table) HeaderRows(count int) error {
	if count < 0 || count > t.native.RowCount() {
		return fmt.Errorf("invalid header row count")
	}
	for i, row := range t.native.Rows() {
		if err := row.SetRepeatAsHeader(i < count); err != nil {
			return err
		}
	}
	return nil
}

// SetMargins replaces table-wide cell padding. Per-cell and per-row padding can
// be supplied through CellMargins; no inherited value is expanded implicitly.
func (t *Table) SetMargins(m Margins) error {
	for _, v := range []int{m.Top, m.Left, m.Bottom, m.Right} {
		if v < 0 || v > 31680 {
			return fmt.Errorf("invalid cell margins")
		}
	}
	t.options.Margins = &m
	return nil
}

// CellMargins adds an explicit padding override for one cell. Negative indices
// fail; merged-cell overrides apply to the physical cell containing that position.
func (t *Table) CellMargins(row, column int, m Margins) error {
	if _, err := t.native.Row(row); err != nil {
		return err
	}
	r, _ := t.native.Row(row)
	if _, err := r.Cell(column); err != nil {
		return err
	}
	for _, v := range []int{m.Top, m.Left, m.Bottom, m.Right} {
		if v < 0 || v > 31680 {
			return fmt.Errorf("invalid cell margins")
		}
	}
	if t.cellMargins == nil {
		t.cellMargins = map[[2]int]Margins{}
	}
	t.cellMargins[[2]int{row, column}] = m
	return nil
}
