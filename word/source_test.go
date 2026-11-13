package word

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcuoli/go-document-docx/publication"
)

type sourceFunc func(context.Context, io.Writer) error

func (f sourceFunc) Write(ctx context.Context, w io.Writer) error { return f(ctx, w) }

func TestUpdatedSourcePublicationAndCleanup(t *testing.T) {
	for _, mode := range []string{"success", "source-failure", "refresh-failure", "cancelled", "writer-failure"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			destination := filepath.Join(dir, "report.docx")
			if err := os.WriteFile(destination, []byte("original"), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			failure := errors.New("sentinel")
			called := false
			var stage string
			source := updatedSource{
				options: Options{TemporaryDirectory: dir},
				source: sourceFunc(func(_ context.Context, w io.Writer) error {
					if _, err := w.Write([]byte("source")); err != nil {
						return err
					}
					if mode == "source-failure" {
						return failure
					}
					return nil
				}),
				refresh: func(_ context.Context, path string) error {
					called = true
					stage = path
					if path == destination {
						t.Fatal("opened destination instead of staging")
					}
					got, err := os.ReadFile(path)
					if err != nil || string(got) != "source" {
						t.Fatalf("%q %v", got, err)
					}
					if mode == "refresh-failure" {
						return failure
					}
					if mode == "cancelled" {
						cancel()
						return nil
					}
					return os.WriteFile(path, []byte("refreshed"), 0600)
				},
			}
			if mode == "writer-failure" {
				err := source.Write(ctx, sourceFuncWriter{failure})
				if !errors.Is(err, failure) {
					t.Fatal(err)
				}
			} else {
				receipt, err := publication.Save(ctx, destination, true, source)
				if mode == "success" {
					if err != nil || !receipt.Installed {
						t.Fatal(receipt, err)
					}
				} else {
					if receipt.Installed || err == nil {
						t.Fatal(receipt, err)
					}
					if mode == "cancelled" && !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
					if mode != "cancelled" && !errors.Is(err, failure) {
						t.Fatal(err)
					}
				}
			}
			if called == (mode == "source-failure") {
				t.Fatal("incorrect refresher invocation", called)
			}
			got, err := os.ReadFile(destination)
			if err != nil {
				t.Fatal(err)
			}
			want := "original"
			if mode == "success" {
				want = "refreshed"
			}
			if string(got) != want {
				t.Fatalf("destination %q, want %q", got, want)
			}
			if stage != "" {
				if _, err := os.Stat(stage); !os.IsNotExist(err) {
					t.Fatal("stage remains", err)
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 {
				t.Fatal("temporary files leaked", entries, err)
			}
		})
	}
}

type sourceFuncWriter struct{ err error }

func (w sourceFuncWriter) Write([]byte) (int, error) { return 0, w.err }

func TestCancelledOrNilSourceWritesNothing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	err := WithUpdatedFields(publication.FromBytes([]byte("source")), Options{}).Write(ctx, &out)
	if !errors.Is(err, context.Canceled) || out.Len() != 0 {
		t.Fatal(err, out.Len())
	}
	err = WithUpdatedFields(nil, Options{}).Write(context.Background(), &out)
	if err == nil || out.Len() != 0 {
		t.Fatal(err, out.Len())
	}
}

func TestRelativeStageDirectoryUsesAbsoluteWordPath(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(cwd, dir)
	if err != nil {
		t.Skip("test temp and checkout are on different volumes")
	}
	source := updatedSource{
		source:  publication.FromBytes([]byte("input")),
		options: Options{TemporaryDirectory: relative},
		refresh: func(_ context.Context, path string) error {
			if !filepath.IsAbs(path) {
				t.Fatalf("Word path is relative: %q", path)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "input" {
				t.Fatalf("%q %v", data, err)
			}
			return os.WriteFile(path, []byte("updated"), 0600)
		},
	}
	var out bytes.Buffer
	if err := source.Write(t.Context(), &out); err != nil || out.String() != "updated" {
		t.Fatal(err, out.String())
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatal("stage leaked", entries, err)
	}
}
