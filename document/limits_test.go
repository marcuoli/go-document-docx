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
	if err := writeTable(context.Background(), &b, Table{Headers: []string{"field"}, Widths: []int{1000}}); err != nil {
		t.Fatal(err)
	}
	x := b.String()
	if strings.Index(x, "<w:tblBorders") > strings.Index(x, "<w:tblLayout") || strings.Index(x, "<w:tblLayout") > strings.Index(x, "<w:tblCellMar") {
		t.Fatal("table properties violate schema order")
	}
}
