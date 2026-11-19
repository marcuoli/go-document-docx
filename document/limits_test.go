package document

import (
	"bytes"
	"context"
	"errors"
	"math"
	"strings"
	"testing"
)

func TestExpansionLimitBeforeLargeBlockCompletes(t *testing.T) {
	var b strings.Builder
	text := strings.Repeat("\t", 5<<20)
	if err := writeText(context.Background(), &b, text); err == nil {
		t.Fatal("expanded XML limit not enforced")
	}
	if b.Len() > (128<<20)+128 {
		t.Fatalf("XML limit overshoot: %d", b.Len())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var cancelled strings.Builder
	if err := writeText(ctx, &cancelled, text); !errors.Is(err, context.Canceled) {
		t.Fatal("text expansion ignored cancellation")
	}
}

func TestZeroValueAndOverflowGeometryWriteNothing(t *testing.T) {
	var d Document
	var out bytes.Buffer
	if err := d.Write(context.Background(), &out); err == nil || out.Len() != 0 {
		t.Fatal("zero-value document emitted invalid geometry")
	}
	o := Options{PageWidth: 11906, PageHeight: 16838, Margin: math.MaxInt, Font: "Arial", HeadingFont: "Arial", SubheadingFont: "Arial", BodySize: 19, HeadingSize: 22, SubheadingSize: 20, TableSize: 16}
	if _, err := New(o); err == nil {
		t.Fatal("overflow margin admitted")
	}
}

func TestGeneratedPropertyOrdering(t *testing.T) {
	d, _ := New(Options{PageWidth: 11906, PageHeight: 16838, Margin: 1080, Font: "Arial", HeadingFont: "Arial", SubheadingFont: "Arial", BodySize: 19, HeadingSize: 22, SubheadingSize: 20, TableSize: 16})
	styles := d.styles()
	if strings.Index(styles, "<w:widowControl") > strings.Index(styles, "<w:spacing") {
		t.Fatal("default paragraph properties violate schema order")
	}
	var b strings.Builder
	bookmarkID := bookmarkState{}
	if err := writeTable(context.Background(), &b, Table{Headers: []string{"field"}, Widths: []int{1000}}, d.options, true, false, &bookmarkID); err != nil {
		t.Fatal(err)
	}
	x := b.String()
	if strings.Index(x, "<w:tblBorders") > strings.Index(x, "<w:tblLayout") || strings.Index(x, "<w:tblLayout") > strings.Index(x, "<w:tblCellMar") {
		t.Fatal("table properties violate schema order")
	}
}

// One paragraph can own many destinations; XML expansion and cancellation must
// be checked while emitting them rather than after the entire paragraph.
func TestQueuedBookmarksBoundExpansionAndCancellation(t *testing.T) {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", (128<<20)-120))
	err := writeParagraph(t.Context(), &b, "Normal", "", "", false, false, 1, "first", "second", "third")
	if err == nil || b.Len() > 128<<20 {
		t.Fatalf("bookmark expansion escaped budget: err=%v bytes=%d", err, b.Len())
	}

	// Fill the remaining budget exactly with an end marker, then require both
	// kinds of marker to reject before modifying a full builder.
	end := `<w:bookmarkEnd w:id="1"/>`
	b.WriteString(strings.Repeat(" ", (128<<20)-b.Len()-len(end)))
	if err := writeBookmarkEnd(t.Context(), &b, 1); err != nil || b.Len() != 128<<20 {
		t.Fatal("exact end boundary", err, b.Len())
	}
	if err := writeBookmarkEnd(t.Context(), &b, 2); err == nil || b.Len() != 128<<20 {
		t.Fatal("end overflow", err, b.Len())
	}
	if err := writeBookmarkStart(t.Context(), &b, "last", 2); err == nil || b.Len() != 128<<20 {
		t.Fatal("start overflow", err, b.Len())
	}
	ctx := &bookmarkCancelContext{Context: t.Context(), remaining: 1}
	b.Reset()
	err = writeParagraph(ctx, &b, "Normal", "", "", false, false, 1, "first", "second", "third")
	if !errors.Is(err, context.Canceled) || strings.Contains(b.String(), `w:name="third"`) {
		t.Fatalf("bookmark loop ignored cancellation: %v %s", err, b.String())
	}
}

type bookmarkCancelContext struct {
	context.Context
	remaining int
}

func (c *bookmarkCancelContext) Err() error {
	if c.remaining == 0 {
		return context.Canceled
	}
	c.remaining--
	return nil
}
func TestQueuedBookmarkValidationCancellation(t *testing.T) {
	for _, validate := range []struct {
		name string
		run  func(context.Context, *Document) error
	}{
		{"text", func(ctx context.Context, d *Document) error { return d.validate(ctx) }},
		{"names", func(ctx context.Context, d *Document) error { _, err := d.navigationNames(ctx); return err }},
	} {
		t.Run(validate.name, func(t *testing.T) {
			d, _ := New(Options{PageWidth: 11906, PageHeight: 16838, Margin: 1080, Font: "Arial", HeadingFont: "Arial", SubheadingFont: "Arial", BodySize: 19, HeadingSize: 22, SubheadingSize: 20, TableSize: 16})
			d.BookmarkNext("first")
			d.BookmarkNext("second")
			d.BookmarkNext("third")
			d.Paragraph("")
			ctx := &bookmarkCancelContext{Context: t.Context(), remaining: 2}
			if err := validate.run(ctx, d); !errors.Is(err, context.Canceled) {
				t.Fatal("validation did not stop within one paragraph", err)
			}
		})
	}
}
