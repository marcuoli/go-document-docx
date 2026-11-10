package editing

import (
	"fmt"
	"github.com/mmonterroca/docxgo/v2/domain"
)

// ColumnStyle applies explicit properties and optional fill to one logical column.
func (t *Table) ColumnStyle(column int, style TextStyle, fill string) error {
	if column < 0 || column >= t.native.ColumnCount() {
		return fmt.Errorf("column out of range")
	}
	for row := range t.native.Rows() {
		if err := t.CellStyle(row, column, style, fill); err != nil {
			return err
		}
	}
	return nil
}

// RowMargins applies an explicit padding override to every cell in one row.
func (t *Table) RowMargins(row int, margins Margins) error {
	native, err := t.native.Row(row)
	if err != nil {
		return err
	}
	for column := range native.Cells() {
		if err := t.CellMargins(row, column, margins); err != nil {
			return err
		}
	}
	return nil
}

// Normal appends justified body prose using the report family, size and black
// text, with six-point paragraph spacing. Low-level Paragraph can inherit a
// caller template instead or explicitly override individual zero-valued fields.
func (d *Document) Normal(text string) (*Paragraph, error) {
	alignment := domain.AlignmentJustify
	space := 120
	return d.Paragraph(text, ParagraphOptions{Alignment: &alignment, Before: &space, After: &space, Text: TextStyle{Font: d.options.Font, Size: d.options.BodySize, Color: "000000"}})
}
