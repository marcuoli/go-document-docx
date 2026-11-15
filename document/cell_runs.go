package document

// CellRun contains literal cell text and an optional six-digit RGB TextColor
// without a leading #. Empty color inherits Cell.TextColor or the paragraph
// style; Cell.Bold applies to every run. Text preserves spaces, tabs and valid
// Unicode; CRLF/CR become visual line breaks like LF. Runs never interpret labels
// or execute text. AddTable copies their slices and Write validates all content
// against the existing document text/XML limits before writing output.
type CellRun struct {
	// Text is a literal span; line-break normalization also crosses run boundaries.
	Text string
	// TextColor overrides the cell/style with six RGB digits; empty inherits.
	TextColor string
}
