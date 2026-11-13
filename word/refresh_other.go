//go:build !windows

package word

import "context"

func refreshFile(ctx context.Context, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrUnavailable
}
