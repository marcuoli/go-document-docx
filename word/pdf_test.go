package word

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/publication"
)

func TestPDFSourcePublicationAndCleanup(t *testing.T) {
	const pdf = "%PDF-1.7\n%%EOF\n"
	failure := errors.New("export or source failure")
	for _, mode := range []string{"success", "source-failure", "export-failure", "bad-output", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			dest := filepath.Join(dir, "report.pdf")
			if err := os.WriteFile(dest, []byte("prior"), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			called := false
			source := pdfSource{
				options: Options{TemporaryDirectory: dir},
				source: sourceFunc(func(_ context.Context, w io.Writer) error {
					if _, err := io.WriteString(w, "finalized input"); err != nil {
						return err
					}
					if mode == "source-failure" {
						return failure
					}
					return nil
				}),
				export: func(_ context.Context, input, output string) error {
					called = true
					if !filepath.IsAbs(input) || filepath.Dir(input) == dir ||
						filepath.Dir(input) != filepath.Dir(output) || output == dest {
						t.Fatalf("export paths are not privately staged: %q, %q", input, output)
					}
					got, err := os.ReadFile(input)
					if err != nil || string(got) != "finalized input" {
						t.Fatalf("input = %q, err = %v", got, err)
					}
					data := pdf
					if mode == "bad-output" {
						data = "not a PDF"
					}
					if err := os.WriteFile(output, []byte(data), 0600); err != nil {
						return err
					}
					switch mode {
					case "export-failure":
						return failure
					case "cancelled":
						cancel()
					}
					return nil
				},
			}
			receipt, err := publication.Save(ctx, dest, true, source)
			want := "prior"
			if mode == "success" {
				want = pdf
				if err != nil || !receipt.Installed || receipt.Size != int64(len(pdf)) {
					t.Fatal(receipt, err)
				}
			} else if receipt.Installed || err == nil {
				t.Fatal(receipt, err)
			}
			switch mode {
			case "source-failure", "export-failure":
				if !errors.Is(err, failure) {
					t.Fatal(err)
				}
			case "cancelled":
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			case "bad-output":
				if !strings.Contains(err.Error(), "did not produce a PDF") {
					t.Fatal(err)
				}
			}
			if called != (mode != "source-failure") {
				t.Fatalf("export called = %t", called)
			}
			got, err := os.ReadFile(dest)
			if err != nil || string(got) != want {
				t.Fatalf("destination = %q, want %q, err = %v", got, want, err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 || entries[0].Name() != "report.pdf" {
				t.Fatal("staging remains", entries, err)
			}
		})
	}
}

func TestPDFStreamCancellationAndWriterFailure(t *testing.T) {
	for _, cancelDuringWrite := range []bool{false, true} {
		ctx, cancel := context.WithCancel(t.Context())
		dir := t.TempDir()
		failure := errors.New("writer failure")
		s := pdfSource{
			source:  publication.FromBytes([]byte("input")),
			options: Options{TemporaryDirectory: dir},
			export: func(_ context.Context, _, output string) error {
				// More than one buffer ensures a second read is needed.
				data := "%PDF-1.7\n" + strings.Repeat("a", 80<<10)
				return os.WriteFile(output, []byte(data), 0600)
			},
		}
		want := failure
		written := 0
		err := s.Write(ctx, pdfTestWriter(func(p []byte) (int, error) {
			if !cancelDuringWrite {
				return 0, failure
			}
			written += len(p)
			cancel()
			return len(p), nil
		}))
		if cancelDuringWrite {
			want = context.Canceled
			if written == 0 || written >= 80<<10 {
				t.Fatalf("wrote %d bytes while cancelling", written)
			}
		}
		cancel()
		if !errors.Is(err, want) {
			t.Fatalf("cancel mode %t: %v, want %v", cancelDuringWrite, err, want)
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			t.Fatal("staging remains", entries, err)
		}
	}
}

type pdfTestWriter func([]byte) (int, error)

func (f pdfTestWriter) Write(p []byte) (int, error) { return f(p) }
