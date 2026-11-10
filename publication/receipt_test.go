package publication_test

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcuoli/go-document-docx/publication"
)

type sourceFunc func(context.Context, io.Writer) error

func (f sourceFunc) Write(ctx context.Context, w io.Writer) error { return f(ctx, w) }
func TestCommittedReceiptAndFailedSourcePreserveDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.docx")
	payload := []byte("Exact generated document bytes")
	receipt, err := publication.Save(context.Background(), path, false, sourceFunc(func(_ context.Context, w io.Writer) error { _, err := w.Write(payload); return err }))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha512.Sum512(payload)
	if !receipt.Installed || receipt.Size != int64(len(payload)) || receipt.SHA512 != hex.EncodeToString(sum[:]) {
		t.Fatal(receipt)
	}
	failure := errors.New("source failed")
	receipt, err = publication.Save(context.Background(), path, true, sourceFunc(func(_ context.Context, w io.Writer) error { _, _ = w.Write([]byte("partial")); return failure }))
	if !errors.Is(err, failure) || receipt.Installed || receipt.Size != 0 || receipt.SHA512 != "" {
		t.Fatal(receipt, err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != string(payload) {
		t.Fatal("destination replaced on failure")
	}
	files, _ := os.ReadDir(filepath.Dir(path))
	if len(files) != 1 {
		t.Fatal("staging leaked", files)
	}
}
