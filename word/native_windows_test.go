//go:build windows && integration

package word

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/publication"
	"github.com/marcuoli/go-document-docx/report"
)

func TestNativeWordPopulatesTOC(t *testing.T) {
	if os.Getenv("DOCX_WORD_INTEGRATION") != "1" {
		t.Skip("explicit desktop Word integration opt-in required")
	}
	d, err := document.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	d.Heading("First synthetic section", 1)
	d.Paragraph("First page.")
	d.HeadingOnNewPage("Second synthetic section", 1)
	d.Paragraph("Second page.")
	var original bytes.Buffer
	if err = d.Write(context.Background(), &original); err != nil {
		t.Fatal(err)
	}
	input, err := zip.NewReader(bytes.NewReader(original.Bytes()), int64(original.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var fixture bytes.Buffer
	output := zip.NewWriter(&fixture)
	for _, f := range input.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if f.Name == "word/document.xml" {
			marker := []byte("<w:body>")
			toc := []byte("<w:body><w:p><w:r><w:fldChar w:fldCharType=\"begin\"/></w:r><w:r><w:instrText xml:space=\"preserve\"> TOC \\o &quot;1-3&quot; \\h \\z \\u </w:instrText></w:r><w:r><w:fldChar w:fldCharType=\"separate\"/></w:r><w:r><w:t>Unpopulated contents</w:t></w:r><w:r><w:fldChar w:fldCharType=\"end\"/></w:r></w:p>")
			if !bytes.Contains(data, marker) {
				t.Fatal("fixture body missing")
			}
			toc = append(toc, []byte("<w:p><w:r><w:fldChar w:fldCharType=\"begin\"/></w:r><w:r><w:instrText xml:space=\"preserve\"> TOC \\c &quot;Table&quot; </w:instrText></w:r><w:r><w:fldChar w:fldCharType=\"separate\"/></w:r><w:r><w:t>Stale caption entry</w:t></w:r><w:r><w:fldChar w:fldCharType=\"end\"/></w:r></w:p>")...)
			data = bytes.Replace(data, marker, toc, 1)
		}
		w, err := output.Create(f.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err = output.Close(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "updated.docx")
	receipt, err := publication.Save(context.Background(), path, false, WithUpdatedFields(publication.FromBytes(fixture.Bytes()), Options{TemporaryDirectory: dir}))
	if err != nil || !receipt.Installed {
		t.Fatal(receipt, err)
	}
	result, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer result.Close()
	found := false
	for _, f := range result.File {
		if f.Name != "word/document.xml" {
			continue
		}
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if strings.Contains(text, "Unpopulated contents") || strings.Contains(text, "Stale caption entry") {
			t.Fatal("TOC placeholder remains")
		}
		// Each heading must appear both as an entry and in the document body.
		for _, title := range []string{"First synthetic section", "Second synthetic section"} {
			if strings.Count(text, title) < 2 {
				t.Fatalf("TOC entry absent: %s", title)
			}
		}
		if !strings.Contains(text, "PAGEREF") && !strings.Contains(text, "_Toc") {
			t.Fatal("TOC page references absent")
		}
		found = true
	}
	if !found {
		t.Fatal("document part absent")
	}
}
