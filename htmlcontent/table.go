// Package htmlcontent imports a bounded, explicit subset of HTML document content.
// It uses the maintained HTML parser; it is not a CSS layout engine and never
// fetches external resources. Ambiguous table shapes return errors, not stale output.
package htmlcontent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/tabledata"
	"golang.org/x/net/html"
)

// ErrNotFound means the requested table was absent. The caller owns the resulting
// evidence/coverage message; no replacement table or successful assessment is inferred.
var ErrNotFound = errors.New("HTML table not found")

// Selector chooses a unique table by ID or summary. Empty selects the only table.
type Selector struct{ ID, Summary string }

// TableOptions controls explicit column selection and header handling. Headerless
// input requires caller-supplied headers. NullTokens is opt-in and case-sensitive.
// Limit counts body rows. Nil preserves all rows.
type TableOptions struct {
	Select                                         []int
	SelectNames                                    []string
	HeaderlessHeaders, OverrideHeaders, NullTokens []string
	Limit                                          *int
	BackgroundAttribute, ForegroundAttribute       string
}

// Table contains parsed literal data and per-cell colors, including row-zero
// header colors. Cells are addressed after selection/limit/override operations.
type Table struct {
	Data   tabledata.Data
	Colors map[[2]int]Colors
}

// Colors contains explicit six-digit RGB values; an empty value inherits.
type Colors struct{ Background, Foreground string }

// ReadTable parses one unambiguous rectangular HTML table. Spans and nested tables
// are unsupported; malformed/ragged input fails as a whole without dropping rows.
func ReadTable(ctx context.Context, reader io.Reader, selector Selector, options TableOptions) (Table, error) {
	root, err := parse(ctx, reader)
	if err != nil {
		return Table{}, err
	}
	if selector.ID != "" && selector.Summary != "" {
		return Table{}, fmt.Errorf("select by ID or summary")
	}
	matches := []*html.Node{}
	walk(root, func(n *html.Node) {
		if n.Type != html.ElementNode || n.Data != "table" {
			return
		}
		if selector.ID != "" && attr(n, "id") != selector.ID {
			return
		}
		if selector.Summary != "" && attr(n, "summary") != selector.Summary {
			return
		}
		matches = append(matches, n)
	})
	if len(matches) == 0 {
		return Table{}, ErrNotFound
	}
	if len(matches) != 1 {
		return Table{}, fmt.Errorf("ambiguous HTML table selection")
	}
	return readTable(matches[0], options)
}

// AppendTable appends parsed values and their explicit colors. A failed parse
// never returns an older table handle. Caller metadata controls width/formatting.
func AppendTable(doc *editing.Document, table Table, options editing.TableOptions) (*editing.Table, error) {
	t, err := doc.Table(table.Data, options)
	if err != nil {
		return nil, err
	}
	for key, colors := range table.Colors {
		if err := t.CellStyle(key[0], key[1], editing.TextStyle{Color: colors.Foreground}, colors.Background); err != nil {
			return nil, err
		}
	}
	return t, nil
}

func readTable(table *html.Node, options TableOptions) (Table, error) {
	rows := [][]*html.Node{}
	nested := false
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n != table && n.Type == html.ElementNode && n.Data == "table" {
			nested = true
			return
		}
		if n.Type == html.ElementNode && n.Data == "tr" {
			cells := []*html.Node{}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
					cells = append(cells, c)
				}
			}
			if len(cells) > 0 {
				rows = append(rows, cells)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(table)
	if nested {
		return Table{}, fmt.Errorf("nested HTML tables are unsupported")
	}
	if len(rows) == 0 {
		return Table{}, fmt.Errorf("HTML table has no rows")
	}
	count := len(rows[0])
	if count > 32 {
		return Table{}, fmt.Errorf("HTML table exceeds 32 columns")
	}
	if len(rows) > 1_000_000/count {
		return Table{}, fmt.Errorf("HTML table exceeds one million cells")
	}
	headerless := len(options.HeaderlessHeaders) > 0
	if !headerless {
		for _, cell := range rows[0] {
			if cell.Data != "th" {
				return Table{}, fmt.Errorf("headerless HTML table needs explicit headers")
			}
		}
	} else if len(options.HeaderlessHeaders) != count {
		return Table{}, fmt.Errorf("header schema does not match rows")
	}
	selected := options.Select
	if len(options.SelectNames) > 0 {
		if len(selected) > 0 {
			return Table{}, fmt.Errorf("select columns by names or indices")
		}
		headers := append([]string(nil), options.HeaderlessHeaders...)
		if !headerless {
			headers = make([]string, count)
			for i, cell := range rows[0] {
				headers[i] = strings.TrimSpace(text(cell))
			}
		}
		source := tabledata.Data{Headers: headers}
		for _, name := range options.SelectNames {
			column, err := source.Column(name)
			if err != nil {
				return Table{}, err
			}
			selected = append(selected, column)
		}
	}
	if len(selected) == 0 {
		selected = make([]int, count)
		for i := range selected {
			selected[i] = i
		}
	}
	seen := map[int]bool{}
	for _, c := range selected {
		if c < 0 || c >= count || seen[c] {
			return Table{}, fmt.Errorf("invalid or duplicate selected column")
		}
		seen[c] = true
	}
	output := Table{Colors: map[[2]int]Colors{}}
	if headerless {
		for _, c := range selected {
			output.Data.Headers = append(output.Data.Headers, options.HeaderlessHeaders[c])
		}
	}
	background, foreground := options.BackgroundAttribute, options.ForegroundAttribute
	if background == "" {
		background = "bgcolor"
	}
	if foreground == "" {
		foreground = "color"
	}
	for r, row := range rows {
		if len(row) != count {
			return Table{}, fmt.Errorf("ragged HTML table row %d", r)
		}
		values := []string{}
		for c, cell := range row {
			for _, name := range []string{"colspan", "rowspan"} {
				if value := attr(cell, name); value != "" && value != "1" {
					return Table{}, fmt.Errorf("HTML cell spans are unsupported")
				}
			}
			if !seen[c] {
				continue
			}
		}
		for c, source := range selected {
			cell := row[source]
			value := strings.TrimSpace(text(cell))
			for _, null := range options.NullTokens {
				if value == null {
					value = ""
				}
			}
			values = append(values, value)
			color := Colors{Background: rgb(attr(cell, background)), Foreground: rgb(attr(cell, foreground))}
			if attr(cell, background) != "" && color.Background == "" || attr(cell, foreground) != "" && color.Foreground == "" {
				return Table{}, fmt.Errorf("unsupported HTML RGB color")
			}
			if color.Background != "" || color.Foreground != "" {
				index := r
				if headerless {
					index++
				}
				output.Colors[[2]int{index, c}] = color
			}
		}
		if r == 0 && !headerless {
			output.Data.Headers = values
		} else {
			output.Data.Rows = append(output.Data.Rows, values)
		}
	}
	if len(options.OverrideHeaders) > 0 {
		if len(options.OverrideHeaders) != len(output.Data.Headers) {
			return Table{}, fmt.Errorf("override headers must match selected columns")
		}
		output.Data.Headers = append([]string(nil), options.OverrideHeaders...)
	}
	if options.Limit != nil {
		if *options.Limit < 0 {
			return Table{}, fmt.Errorf("negative HTML row limit")
		}
		output.Data.Rows = output.Data.Rows[:min(*options.Limit, len(output.Data.Rows))]
		for key := range output.Colors {
			if key[0] > len(output.Data.Rows) {
				delete(output.Colors, key)
			}
		}
	}
	if _, err := output.Data.Clone(); err != nil {
		return Table{}, err
	}
	return output, nil
}

func parse(ctx context.Context, reader io.Reader) (*html.Node, error) {
	if reader == nil {
		return nil, fmt.Errorf("HTML reader is required")
	}
	var b strings.Builder
	buffer := make([]byte, 32<<10)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		n, err := reader.Read(buffer)
		if n > 0 {
			if b.Len()+n > 8<<20 {
				return nil, fmt.Errorf("HTML exceeds 8 MiB")
			}
			b.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, io.ErrNoProgress
		}
	}
	if !utf8.ValidString(b.String()) {
		return nil, fmt.Errorf("HTML is not UTF-8")
	}
	return html.Parse(strings.NewReader(b.String()))
}
func walk(n *html.Node, visit func(*html.Node)) {
	visit(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, visit)
	}
}
func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func text(n *html.Node) string {
	var b strings.Builder
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		if n.Type == html.ElementNode && n.Data == "br" {
			b.WriteByte('\n')
		}
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	return b.String()
}
func rgb(s string) string {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return ""
	}
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return ""
		}
	}
	return strings.ToUpper(s)
}
