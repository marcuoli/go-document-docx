package word

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/marcuoli/go-document-docx/publication"
)

// PDF wraps a finalized DOCX source for export with desktop Microsoft Word.
// On Write it stages the input on disk, opens the copy read-only, exports all
// pages with heading bookmarks, and streams the resulting PDF to the writer.
// Automatic literal cell ScreenTip links to their generated self-anchors
// are omitted from the PDF when the SDK producer marker is present.
// The source and writer remain caller-owned. Only the private staging files
// are modified, and PDF adds neither a field update nor a DOCX save.
// Use publication.Save to preserve the destination if export or streaming fails.
//
// Word requires Windows and a configured interactive desktop Word profile.
// Calls share the Word automation lock with WithUpdatedFields. Cancellation is
// checked between COM calls and during PDF streaming. An in-flight Word call
// cannot be interrupted. The temporary directory must have room for the staged
// DOCX and PDF.
func PDF(source publication.Source, options Options) publication.Source {
	return pdfSource{source: source, options: options, export: func(ctx context.Context, input, output string) error {
		return exportPDFFile(ctx, input, output, true)
	}}
}

// PDFPreservingLinks exports a staged DOCX using Word's native link handling.
// Use it when the caller has prepared the PDF link targets before export or
// needs the generated cell links in the intermediate PDF. Word may omit or
// reinterpret link metadata, including ScreenTip text. Staging, ownership,
// cancellation and platform requirements match PDF.
func PDFPreservingLinks(source publication.Source, options Options) publication.Source {
	return pdfSource{source: source, options: options, export: func(ctx context.Context, input, output string) error {
		return exportPDFFile(ctx, input, output, false)
	}}
}

type pdfSource struct {
	source  publication.Source
	options Options
	export  func(context.Context, string, string) error
}

func (s pdfSource) Write(ctx context.Context, w io.Writer) (returnedErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.source == nil {
		return errors.New("document source is required")
	}
	base := s.options.TemporaryDirectory
	if base == "" {
		base = os.TempDir()
	}
	base, err := filepath.Abs(base)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp(base, ".word-pdf-*")
	if err != nil {
		return err
	}
	defer func() { returnedErr = errors.Join(returnedErr, removePDFStage(dir)) }()
	input := filepath.Join(dir, "input.docx")
	output := filepath.Join(dir, "output.pdf")
	if err := stagePDFInput(ctx, s.source, input); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.export(ctx, input, output); err != nil {
		return fmt.Errorf("export Word PDF: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := os.Open(output)
	if err != nil {
		return err
	}
	defer func() { returnedErr = errors.Join(returnedErr, f.Close()) }()
	reader := bufio.NewReader(pdfReader{ctx: ctx, file: f})
	const signature = "%PDF-"
	prefix, err := reader.Peek(len(signature))
	if err != nil {
		return fmt.Errorf("read Word PDF header: %w", err)
	}
	if string(prefix) != signature {
		return errors.New("Word export did not produce a PDF")
	}
	if _, err := io.Copy(w, reader); err != nil {
		return err
	}
	return ctx.Err()
}

func stagePDFInput(ctx context.Context, source publication.Source, path string) (returnedErr error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() { returnedErr = errors.Join(returnedErr, f.Close()) }()
	return source.Write(ctx, f)
}

type pdfReader struct {
	ctx  context.Context
	file *os.File
}

func (r pdfReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.file.Read(p)
}
