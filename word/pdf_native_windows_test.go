//go:build windows && integration

package word

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/publication"
	"github.com/marcuoli/go-document-docx/report"
)

func TestNativeWordExportsPDF(t *testing.T) {
	if os.Getenv("DOCX_WORD_INTEGRATION") != "1" {
		t.Skip("explicit desktop Word integration opt-in required")
	}
	doc, err := document.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	doc.Heading("First export section", 1)
	doc.Paragraph("First page.")
	doc.HeadingOnNewPage("Second export section", 1)
	doc.Paragraph("Second page.")
	dir := t.TempDir()
	input := filepath.Join(dir, "finalized.docx")
	output := filepath.Join(dir, "converted.pdf")
	if _, err := publication.Save(t.Context(), input, false, doc); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	source := sourceFunc(func(_ context.Context, w io.Writer) (returnedErr error) {
		f, err := os.Open(input)
		if err != nil {
			return err
		}
		defer func() { returnedErr = errors.Join(returnedErr, f.Close()) }()
		_, err = io.Copy(w, f)
		return err
	})
	receipt, err := publication.Save(t.Context(), output, false, PDF(source, Options{TemporaryDirectory: dir}))
	if err != nil || !receipt.Installed {
		t.Fatal(receipt, err)
	}
	after, err := os.ReadFile(input)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("export changed finalized input", err)
	}
	pdf, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Size != int64(len(pdf)) || !bytes.HasPrefix(pdf, []byte("%PDF-")) ||
		!bytes.Contains(pdf, []byte("/Outlines")) || !bytes.HasSuffix(bytes.TrimSpace(pdf), []byte("%%EOF")) {
		t.Fatalf("expected a committed PDF with outlines, got %d bytes", len(pdf))
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 2 {
		t.Fatal("staging remains", entries, err)
	}
}
