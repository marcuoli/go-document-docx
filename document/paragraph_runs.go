package document

import "slices"

// ParagraphRun is literal text with an optional color for that run only.
type ParagraphRun struct {
	// Text preserves whitespace, tabs and soft breaks as in Document.Paragraph.
	Text string
	// TextColor is six hexadecimal RGB digits without #. Empty inherits the paragraph style.
	TextColor string
}

// ParagraphRuns appends one body paragraph with the given ordered runs. It
// takes a snapshot of the slice. With no runs it appends an empty paragraph.
// Like Paragraph it consumes pending BookmarkNext requests. Write checks text,
// RGB colors and document limits before writing the DOCX.
func (d *Document) ParagraphRuns(runs ...ParagraphRun) {
	d.appendParagraph(block{style: "Normal", runs: slices.Clone(runs)})
}
