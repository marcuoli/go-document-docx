// Package report provides configurable technical-report presentation defaults.
// Product policy, evidence evaluation, content and legal notices belong to callers.
package report

import "github.com/marcuoli/go-document-docx/document"

// Defaults returns independent A4 report options: 0.75-inch margins, 9.5-point
// body, 11/10-point headings and 8-point tables. Font names are references only;
// readers substitute installed fonts when the requested families are unavailable.
func Defaults() document.Options {
	return document.Options{
		PageWidth: 11906, PageHeight: 16838, Margin: 1080,
		Font: "Oracle Sans", HeadingFont: "Oracle Sans Extra Bold", SubheadingFont: "Oracle Sans Semi Bold",
		FallbackFont: "Arial",
		BodySize:     19, HeadingSize: 22, SubheadingSize: 20, TableSize: 16, PageLabel: "Page",
	}
}
