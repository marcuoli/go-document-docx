// Package tabledata transforms owned table values before document rendering.
// Text stays exact unless the caller explicitly requests a display format.
package tabledata

import (
	"fmt"
	"slices"
)

// Data contains one header row and rectangular body rows. Methods return owned copies.
type Data struct {
	Headers []string
	Rows    [][]string
}

// Clone validates shape and returns a deep copy without modifying the source.
func (d Data) Clone() (Data, error) {
	if err := d.validate(); err != nil {
		return Data{}, err
	}
	out := Data{Headers: slices.Clone(d.Headers), Rows: make([][]string, len(d.Rows))}
	for i, row := range d.Rows {
		out.Rows[i] = slices.Clone(row)
	}
	return out, nil
}

// Shape validation is shared by transforms that construct their own owned rows;
// validating must not require a full deep copy that is immediately discarded.
func (d Data) validate() error {
	if len(d.Headers) == 0 || len(d.Headers) > 32 {
		return fmt.Errorf("table must have 1 to 32 columns")
	}
	if len(d.Rows) > 1_000_000/len(d.Headers)-1 {
		return fmt.Errorf("table exceeds one million cells")
	}
	for i, row := range d.Rows {
		if len(row) != len(d.Headers) {
			return fmt.Errorf("row %d has %d cells; want %d", i, len(row), len(d.Headers))
		}
	}
	return nil
}

// Select returns columns in the requested order. Duplicate or missing columns fail.
func (d Data) Select(columns []int) (Data, error) {
	if err := d.validate(); err != nil {
		return Data{}, err
	}
	if len(columns) == 0 {
		return Data{}, fmt.Errorf("select at least one column")
	}
	seen := map[int]bool{}
	out := Data{Rows: make([][]string, len(d.Rows))}
	for _, col := range columns {
		if col < 0 || col >= len(d.Headers) || seen[col] {
			return Data{}, fmt.Errorf("invalid or duplicate column %d", col)
		}
		seen[col] = true
		out.Headers = append(out.Headers, d.Headers[col])
	}
	for i, row := range d.Rows {
		for _, col := range columns {
			out.Rows[i] = append(out.Rows[i], row[col])
		}
	}
	return out, nil
}

// Column resolves a unique, exact header name; duplicate names are ambiguous.
func (d Data) Column(name string) (int, error) {
	found := -1
	for i, h := range d.Headers {
		if h == name {
			if found >= 0 {
				return 0, fmt.Errorf("ambiguous column %q", name)
			}
			found = i
		}
	}
	if found < 0 {
		return 0, fmt.Errorf("column %q not found", name)
	}
	return found, nil
}

// Transpose treats the existing header as the first row and promotes the first
// resulting row to headers. The result must still fit the 32-column contract.
func (d Data) Transpose() (Data, error) {
	if err := d.validate(); err != nil {
		return Data{}, err
	}
	if len(d.Rows)+1 > 32 {
		return Data{}, fmt.Errorf("transposed table exceeds 32 columns")
	}
	rows := make([][]string, len(d.Headers))
	for col, header := range d.Headers {
		rows[col] = make([]string, len(d.Rows)+1)
		rows[col][0] = header
		for i, row := range d.Rows {
			rows[col][i+1] = row[col]
		}
	}
	return Data{Headers: rows[0], Rows: rows[1:]}, nil
}

// Limit returns at most count body rows. Zero means no rows, not an implicit default.
func (d Data) Limit(count int) (Data, error) {
	out, err := d.Clone()
	if err != nil {
		return Data{}, err
	}
	if count < 0 {
		return Data{}, fmt.Errorf("negative row limit")
	}
	out.Rows = out.Rows[:min(count, len(out.Rows))]
	return out, nil
}

// Shorten returns a Unicode-aware display value and its unchanged full value.
// The caller must preserve the full value in a tooltip or another readable artifact.
func Shorten(text string, limit int) (display, full string, err error) {
	if limit < 4 {
		return "", "", fmt.Errorf("ellipsis limit must be at least four characters")
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text, text, nil
	}
	left := (limit - 3) / 2
	right := limit - 3 - left
	return string(runes[:left]) + "..." + string(runes[len(runes)-right:]), text, nil
}
