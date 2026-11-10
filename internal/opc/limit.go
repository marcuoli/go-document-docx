package opc

import (
	"context"
	"fmt"
	"io"
)

// Limited bounds serialized output while preserving cancellation and short-write
// errors. The limit applies to compressed bytes, matching Read's admission bound.
func Limited(ctx context.Context, w io.Writer, max int64) io.Writer {
	return &limitedWriter{ctx: ctx, destination: w, max: max}
}

type limitedWriter struct {
	ctx         context.Context
	destination io.Writer
	max, count  int64
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.max-w.count {
		return 0, fmt.Errorf("compressed document exceeds output limit")
	}
	n, err := (Writer{Ctx: w.ctx, Destination: w.destination}).Write(p)
	w.count += int64(n)
	return n, err
}
