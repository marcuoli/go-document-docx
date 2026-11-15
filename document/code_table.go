package document

// CodeTable appends a full-width literal code block with a repeated bold header
// and one body cell. Both use white Consolas text on black at Options.TableSize
// half-points, left aligned with natural wrapping. Fonts are referenced, not
// embedded; glyph coverage and pagination depend on the reader. The body row
// may split across pages; no fixed height, truncation or execution is applied.
//
// Header and code are caller-owned literal strings; empty strings are allowed.
// Spaces, tabs, blank lines and Unicode are retained; CRLF and CR become visual
// line breaks like LF. Write validates UTF-8/XML characters and the document's
// existing aggregate limits (64 MiB text, 128 MiB expanded document XML) before
// sending bytes to its writer. Construction and serialization are in memory;
// callers own source retrieval, completeness, budgets and publication. Like
// other Document methods, CodeTable requires serial use.
func (d *Document) CodeTable(header, code string) {
	d.blocks = append(d.blocks, block{
		table: &Table{
			Headers: []string{header},
			Widths:  []int{d.options.PageWidth - 2*d.options.Margin},
			Rows:    [][]Cell{{{Text: code}}},
		},
		codeTable: true,
	})
}

func (d *Document) hasCodeTables() bool {
	for _, b := range d.blocks {
		if b.codeTable {
			return true
		}
	}
	return false
}
