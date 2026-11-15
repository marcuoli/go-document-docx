package document

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// cancelOnSecondCheck cancels a real context deterministically during the serial
// renderer's second cancellation check, without timers or a racing goroutine.
type cancelOnSecondCheck struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (c *cancelOnSecondCheck) Err() error {
	c.checks++
	if c.checks == 2 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestEmptyCellRunsObserveMidRenderCancellation(t *testing.T) {
	base, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx := &cancelOnSecondCheck{Context: base, cancel: cancel}
	var out strings.Builder
	err := writeCellParagraph(ctx, &out, "TableText", Cell{Runs: []CellRun{{TextColor: "007F00"}, {TextColor: "B3261E"}, {}}}, "")
	if !errors.Is(err, context.Canceled) || ctx.checks != 2 || strings.Count(out.String(), "<w:r>") != 1 {
		t.Fatal("empty runs ignored cancellation between runs", err, ctx.checks)
	}
}
