// Package nativecontent walks supported upstream document objects without creating
// missing headers or footers. It supplies shared read-only traversal for validation
// and template resource binding; it does not interpret report policy.
package nativecontent

import (
	"context"
	"fmt"

	"github.com/mmonterroca/docxgo/v2/domain"
)

// Walk visits represented paragraphs in the body, nested cells and existing
// headers/footers. It never calls lazy Header/Footer getters. The released backend
// exposes copy-returning enumeration methods beyond the general section interface.
// Cells and nesting are bounded; visit must not mutate the containing structure.
func Walk(ctx context.Context, doc domain.Document, visit func(domain.Paragraph) error) error {
	cells := 0
	paragraph := func(p domain.Paragraph) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return visit(p)
	}
	var tables func([]domain.Table, int) error
	tables = func(list []domain.Table, depth int) error {
		if depth > 64 {
			return fmt.Errorf("native table nesting exceeds supported depth")
		}
		for _, table := range list {
			for _, row := range table.Rows() {
				for _, cell := range row.Cells() {
					cells++
					if cells > 1_000_000 {
						return fmt.Errorf("document exceeds one million cells")
					}
					for _, p := range cell.Paragraphs() {
						if err := paragraph(p); err != nil {
							return err
						}
					}
					if err := tables(cell.Tables(), depth+1); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	for _, p := range doc.Paragraphs() {
		if err := paragraph(p); err != nil {
			return err
		}
	}
	if err := tables(doc.Tables(), 0); err != nil {
		return err
	}
	for _, section := range doc.Sections() {
		enumeration, ok := section.(interface {
			HeadersAll() map[domain.HeaderType]domain.Header
			FootersAll() map[domain.FooterType]domain.Footer
		})
		if !ok {
			return fmt.Errorf("upstream header enumeration is unsupported")
		}
		headers, footers := enumeration.HeadersAll(), enumeration.FootersAll()
		for _, kind := range []domain.HeaderType{domain.HeaderDefault, domain.HeaderFirst, domain.HeaderEven} {
			if h := headers[kind]; h != nil {
				for _, p := range h.Paragraphs() {
					if err := paragraph(p); err != nil {
						return err
					}
				}
				if err := tables(h.Tables(), 0); err != nil {
					return err
				}
			}
		}
		for _, kind := range []domain.FooterType{domain.FooterDefault, domain.FooterFirst, domain.FooterEven} {
			if f := footers[kind]; f != nil {
				for _, p := range f.Paragraphs() {
					if err := paragraph(p); err != nil {
						return err
					}
				}
				if err := tables(f.Tables(), 0); err != nil {
					return err
				}
			}
		}
	}
	return ctx.Err()
}
