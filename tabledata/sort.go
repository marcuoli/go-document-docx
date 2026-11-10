package tabledata

import (
	"cmp"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"
)

// Kind identifies an explicit interpretation; values are never inferred from names.
type Kind string

// Text, Number and Date are the supported stable sort interpretations.
const (
	Text   Kind = "text"
	Number Kind = "number"
	Date   Kind = "date"
)

// SortKey selects a column, direction and interpretation. Date requires a Go layout.
// Blank values sort last in either direction. Invalid nonblank values are errors.
type SortKey struct {
	Column     int
	Kind       Kind
	Descending bool
	Layout     string
}

// Sort returns a stable, owned ordering and preserves original cell text, including
// large integers, numeric spellings, and dates. It never coerces missing data to zero.
func (d Data) Sort(keys ...SortKey) (Data, error) {
	out, err := d.Clone()
	if err != nil {
		return Data{}, err
	}
	type value struct {
		blank  bool
		text   string
		number *big.Rat
		date   time.Time
	}
	parsed := make([][]value, len(keys))
	for k, key := range keys {
		if key.Column < 0 || key.Column >= len(d.Headers) {
			return Data{}, fmt.Errorf("sort column out of range")
		}
		if key.Kind != "" && key.Kind != Text && key.Kind != Number && key.Kind != Date {
			return Data{}, fmt.Errorf("unknown sort kind")
		}
		if key.Kind == Date && key.Layout == "" {
			return Data{}, fmt.Errorf("date sort requires a Go layout")
		}
		parsed[k] = make([]value, len(d.Rows))
		for r, row := range d.Rows {
			text := strings.TrimSpace(row[key.Column])
			v := value{blank: text == "", text: text}
			if !v.blank {
				switch key.Kind {
				case Number:
					v.number, err = number(strings.ReplaceAll(text, ",", ""))
					if err != nil {
						return Data{}, fmt.Errorf("row %d is not a number", r)
					}
				case Date:
					v.date, err = time.Parse(key.Layout, text)
					if err != nil {
						return Data{}, fmt.Errorf("row %d date: %w", r, err)
					}
				}
			}
			parsed[k][r] = v
		}
	}
	order := make([]int, len(out.Rows))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		for k, key := range keys {
			x, y := parsed[k][a], parsed[k][b]
			if x.blank != y.blank {
				if x.blank {
					return 1
				}
				return -1
			}
			if x.blank {
				continue
			}
			n := 0
			switch key.Kind {
			case Number:
				n = x.number.Cmp(y.number)
			case Date:
				n = x.date.Compare(y.date)
			default:
				n = cmp.Compare(x.text, y.text)
			}
			if key.Descending {
				n = -n
			}
			if n != 0 {
				return n
			}
		}
		return 0
	})
	for i, original := range order {
		out.Rows[i] = slices.Clone(d.Rows[original])
	}
	return out, nil
}
