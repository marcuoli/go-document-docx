package document

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCellScreenTipBoundedAttributeExpansion(t *testing.T) {
	var out strings.Builder
	err := writeCellParagraph(t.Context(), &out, "TableText", Cell{Text: "✔", ScreenTip: strings.Repeat("&", 27<<20)}, "", 0, 1)
	if err == nil {
		t.Fatal("expanded XML limit was not enforced")
	}
	if out.Len() > (128<<20)+128 {
		t.Fatalf("attribute expansion overshot XML budget: %d", out.Len())
	}
}

func TestCellScreenTipAttributeCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var out strings.Builder
	err := writeCellParagraph(ctx, &out, "TableText", Cell{Text: "✔", ScreenTip: strings.Repeat("&", 1<<20)}, "", 0, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation identity changed", err)
	}
	if out.Len() > 512 {
		t.Fatalf("cancelled tooltip was expanded: %d", out.Len())
	}
}

// attributeCancelContext stops after the first prefix has reached the builder.
type attributeCancelContext struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (c *attributeCancelContext) Err() error {
	c.checks++
	if c.checks == 3 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestCellScreenTipMidAttributeCancellation(t *testing.T) {
	base, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx := &attributeCancelContext{Context: base, cancel: cancel}
	var out strings.Builder
	err := writeScreenTipAttribute(ctx, &out, "prefix "+strings.Repeat("&", 1<<20))
	if !errors.Is(err, context.Canceled) || ctx.checks != 3 || out.String() != "prefix " {
		t.Fatal("attribute expansion ignored deterministic mid-write cancellation", err, ctx.checks, out.String())
	}
}
