// Package editing provides reusable document authoring helpers backed by docxgo.
// It preserves the small literal report API in package document. Native exposes
// the upstream API for additional supported operations without another object model.
package editing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/internal/nativecontent"
	"github.com/marcuoli/go-document-docx/internal/opc"
	docx "github.com/mmonterroca/docxgo/v2"
	"github.com/mmonterroca/docxgo/v2/domain"
)

// Document owns an upstream document and its layout amendments. Use it serially;
// changes through Native share ownership. New/Open copy all caller-provided bytes.
type Document struct {
	native          domain.Document
	options         document.Options
	paragraphs      []*Paragraph
	tables          []*Table
	captionCounters map[string]int
	gutters         []int
	styles          []domain.ParagraphStyle
	// HeadingProgress is an optional caller-owned notification. No global progress state is used.
	HeadingProgress func(level int, text string)
}

// New creates an editor with the same configurable defaults as the literal API.
// A neutral generated template retains font hints and compatibility settings.
func New(options document.Options) (*Document, error) {
	base, err := document.New(options)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err := base.Write(context.Background(), &b); err != nil {
		return nil, err
	}
	return Open(context.Background(), b.Bytes(), options)
}

// Open reads a caller-supplied DOCX after bounded package admission. It does not
// fetch links, discover assets, interpret macros or execute template expressions.
// Options configure new content; they do not replace existing template geometry.
func Open(ctx context.Context, data []byte, options document.Options) (*Document, error) {
	if _, err := document.New(options); err != nil {
		return nil, err
	}
	p, err := opc.Read(ctx, data)
	if err != nil {
		return nil, err
	}
	for name := range p.Parts {
		if strings.HasSuffix(name, ".xml") || strings.HasSuffix(name, ".rels") {
			if _, err := p.XML(name); err != nil {
				return nil, err
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	model, err := docx.OpenDocumentFromBytes(bytes.Clone(data))
	if err != nil {
		return nil, err
	}
	documentXML, err := p.XML("word/document.xml")
	if err != nil {
		return nil, err
	}
	gutters := []int{}
	for _, section := range opc.Elements(documentXML.Root(), opc.Word, "sectPr") {
		gutter := 0
		if margins := opc.Child(section, opc.Word, "pgMar"); margins != nil {
			if text := opc.Attr(margins, opc.Word, "gutter"); text != "" {
				gutter, err = strconv.Atoi(text)
				if err != nil || gutter < 0 || gutter > 31680 {
					return nil, fmt.Errorf("invalid section gutter")
				}
			}
		}
		gutters = append(gutters, gutter)
	}
	return &Document{native: model, options: options, captionCounters: map[string]int{}, gutters: gutters}, nil
}

// Native returns the owned upstream interface. It is not a copy or concurrency
// boundary. Helpers validate that their paragraph/table handles still exist at Write.
func (d *Document) Native() domain.Document { return d.native }

// PageBreak appends a native page break.
func (d *Document) PageBreak() error { return d.native.AddPageBreak() }

// SetMetadata stores caller-authored core properties. It does not infer an author,
// subject, clock, database identity, registration record or generated-file status.
func (d *Document) SetMetadata(metadata domain.Metadata) error {
	for _, value := range append([]string{metadata.Title, metadata.Subject, metadata.Creator, metadata.Description, metadata.Created, metadata.Modified}, metadata.Keywords...) {
		if err := validText(value); err != nil {
			return err
		}
	}
	return d.native.SetMetadata(&metadata)
}

// SetPageLayout updates the existing sections with explicit upstream page units.
// Native permits per-section edits when sections require different geometry.
func (d *Document) SetPageLayout(size domain.PageSize, margins domain.Margins) error {
	for _, section := range d.native.Sections() {
		if err := errors.Join(section.SetPageSize(size), section.SetMargins(margins)); err != nil {
			return err
		}
	}
	return nil
}

// Write validates literal text before serialization, applies the small artifact
// controls missing from the upstream API, and writes a complete package. It never
// closes writer; use publication.PublishSource to preserve an existing destination.
// Failed authoring calls can leave a partially changed editor; discard it on error.
func (d *Document) Write(ctx context.Context, writer io.Writer) error {
	if d == nil || d.native == nil || writer == nil {
		return fmt.Errorf("document and writer are required")
	}
	if err := d.validate(ctx); err != nil {
		return err
	}
	var buffer bytes.Buffer
	if _, err := d.native.WriteTo(opc.Limited(ctx, &buffer, 64<<20)); err != nil {
		return err
	}
	p, err := opc.Read(ctx, buffer.Bytes())
	if err != nil {
		return err
	}
	if err := d.amendStyles(ctx, p); err != nil {
		return err
	}
	if err := d.amend(ctx, p); err != nil {
		return err
	}
	return p.Write(ctx, writer)
}

func validText(text string) error { return opc.Text(text) }
func (d *Document) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	total := 0
	check := func(text string) error {
		if len(text) > 64<<20-total {
			return fmt.Errorf("document text exceeds 64 MiB")
		}
		total += len(text)
		return validText(text)
	}
	if metadata := d.native.Metadata(); metadata != nil {
		for _, text := range append([]string{metadata.Title, metadata.Subject, metadata.Creator, metadata.Description, metadata.Created, metadata.Modified}, metadata.Keywords...) {
			if err := check(text); err != nil {
				return err
			}
		}
	}
	for _, style := range d.native.StyleManager().ListStyles() {
		for _, text := range []string{style.ID(), style.Name(), style.Font().Name, style.Font().EastAsia, style.Font().CS} {
			if err := check(text); err != nil {
				return err
			}
		}
	}
	return nativecontent.Walk(ctx, d.native, func(p domain.Paragraph) error {
		for _, run := range p.Runs() {
			font := run.Font()
			for _, text := range []string{run.Text(), font.Name, font.EastAsia, font.CS} {
				if err := check(text); err != nil {
					return err
				}
			}
			for _, field := range run.Fields() {
				for _, text := range []string{field.Code(), field.Result()} {
					if err := check(text); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}
