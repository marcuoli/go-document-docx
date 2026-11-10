// Package publication installs generated documents through shared atomic file operations.
package publication

import (
	"context"
	"errors"
	"os"

	"github.com/marcuoli/go-atomicfile/atomicfile"
	"github.com/marcuoli/go-document-docx/document"
)

// Publish stages, syncs and atomically installs doc in an existing directory.
// It honors explicit overwrite intent, detects destination changes, and removes
// staging files on failure. Inspect installed even when cleanup returns an error.
// Cancellation before commit preserves the existing destination; a commit already
// in progress cannot be revoked. The document and destination must be used serially.
func Publish(ctx context.Context, path string, overwrite bool, doc *document.Document) (installed bool, returnedErr error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	p, err := atomicfile.NewPublication(path, overwrite)
	if err != nil {
		return false, err
	}
	f, err := p.CreateTemp(".document-*.docx")
	if err != nil {
		return false, err
	}
	closed := false
	defer func() {
		if !closed {
			returnedErr = errors.Join(returnedErr, f.Close())
		}
		if err := os.Remove(f.Name()); err != nil && !os.IsNotExist(err) {
			returnedErr = errors.Join(returnedErr, err)
		}
	}()
	if err = doc.Write(ctx, f); err != nil {
		return false, err
	}
	if err = f.Sync(); err != nil {
		return false, err
	}
	err = f.Close()
	closed = true
	if err != nil {
		return false, err
	}
	if err = ctx.Err(); err != nil {
		return false, err
	}
	return p.Commit(f.Name())
}
