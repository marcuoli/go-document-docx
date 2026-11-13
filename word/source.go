// Package word provides optional desktop Microsoft Word post-processing.
// Normal document authoring and publication do not require this package.
package word

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/marcuoli/go-document-docx/publication"
)

// ErrUnavailable identifies a host without supported Microsoft Word automation.
var ErrUnavailable = errors.New("Microsoft Word automation is unavailable; Windows and desktop Word are required")

// Options controls owned staging for Word post-processing.
type Options struct {
	// TemporaryDirectory selects private staging; empty uses the process temp directory.
	// Callers retain ownership of this directory. Only the created file is removed.
	TemporaryDirectory string
}

// WithUpdatedFields wraps source without modifying it. On Write it stages a DOCX,
// updates existing TOCs and caption indexes with Word, repaginates, refreshes page
// numbers and saves before forwarding bytes. It never creates a missing TOC.
// Use publication.Save to preserve the destination when post-processing fails.
// Word requires Windows and an interactive user profile with Word configured.
// Calls are serialized in this process. Context is checked between synchronous
// COM calls; an in-flight Word call cannot be interrupted or given a hard timeout.
// The wrapper closes only its newly created Word instance, never an active one.
// Sources and writers remain caller-owned and must be used serially.
func WithUpdatedFields(source publication.Source, options Options) publication.Source {
	return updatedSource{source: source, options: options, refresh: refreshFile}
}

type updatedSource struct {
	source  publication.Source
	options Options
	refresh func(context.Context, string) error
}

func (s updatedSource) Write(ctx context.Context, w io.Writer) (returnedErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.source == nil {
		return errors.New("document source is required")
	}
	f, err := os.CreateTemp(s.options.TemporaryDirectory, ".word-fields-*.docx")
	if err != nil {
		return err
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
	if err := s.source.Write(ctx, f); err != nil {
		return err
	}
	err = f.Close()
	closed = true
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	absolutePath, err := filepath.Abs(f.Name())
	if err != nil {
		return err
	}
	if err := s.refresh(ctx, absolutePath); err != nil {
		return fmt.Errorf("refresh Word TOC: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := os.ReadFile(f.Name())
	if err != nil {
		return err
	}
	return publication.FromBytes(data).Write(ctx, w)
}
