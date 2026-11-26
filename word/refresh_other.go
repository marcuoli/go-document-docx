//go:build !windows

package word

import (
	"context"
	"os"
)

func refreshFile(ctx context.Context, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrUnavailable
}

func exportPDFFile(ctx context.Context, _, _ string, _ bool) error {
	return refreshFile(ctx, "")
}

func removePDFStage(path string) error {
	return os.RemoveAll(path)
}
