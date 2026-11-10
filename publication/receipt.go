package publication

import (
	"context"
	"io"
)

// Source writes an owned document without closing its caller's writer. Both
// literal documents and editors implement this interface; publication owns files.
type Source interface {
	Write(context.Context, io.Writer) error
}

// Receipt records a committed document's byte count and SHA-512 digest. Fields
// are populated only when Installed is true, including a later cleanup error.
type Receipt struct {
	Installed bool
	Size      int64
	SHA512    string
}

// PublishSource publishes any Source using the same atomic protocol as Publish.
// It reports an installed sibling even when a subsequent cleanup operation fails.
func PublishSource(ctx context.Context, path string, overwrite bool, source Source) (bool, error) {
	receipt, err := Save(ctx, path, overwrite, source)
	return receipt.Installed, err
}

type countWriter struct {
	writer io.Writer
	count  int64
}

// Write counts bytes actually accepted, including partial writes.
func (w *countWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.count += int64(n)
	return n, err
}
