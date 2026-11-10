// Package publication installs generated documents through shared atomic file operations.
package publication

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"github.com/marcuoli/go-atomicfile/atomicfile"
	"github.com/marcuoli/go-document-docx/document"
)

// Publish stages, syncs and atomically installs doc in an existing directory.
// It honors explicit overwrite intent, detects destination changes, and removes
// staging files on failure. Inspect installed even when cleanup returns an error.
// Cancellation before commit preserves the existing destination; a commit already
// in progress cannot be revoked. The document and destination must be used serially.
func Publish(ctx context.Context, path string, overwrite bool, doc *document.Document) (bool, error) {
	return PublishSource(ctx, path, overwrite, doc)
}

// Save stages, syncs and atomically publishes source, returning its committed
// byte count and SHA-512 digest. Registration, retention and acceptance policy
// belong to the caller. A failure before commit preserves the destination.
func Save(ctx context.Context, path string, overwrite bool, source Source) (receipt Receipt, returnedErr error) {
	if source == nil {
		return receipt, errors.New("document source is required")
	}
	if err := ctx.Err(); err != nil {
		return receipt, err
	}
	p, err := atomicfile.NewPublication(path, overwrite)
	if err != nil {
		return receipt, err
	}
	f, err := p.CreateTemp(".document-*.docx")
	if err != nil {
		return receipt, err
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
	digest := sha512.New()
	count := &countWriter{writer: io.MultiWriter(f, digest)}
	if err := source.Write(ctx, count); err != nil {
		return receipt, err
	}
	if err := f.Sync(); err != nil {
		return receipt, err
	}
	err = f.Close()
	closed = true
	if err != nil {
		return receipt, err
	}
	if err := ctx.Err(); err != nil {
		return receipt, err
	}
	receipt.Installed, err = p.Commit(f.Name())
	if receipt.Installed {
		receipt.Size = count.count
		receipt.SHA512 = hex.EncodeToString(digest.Sum(nil))
	}
	return receipt, err
}
