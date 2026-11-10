package tabledata

import (
	"reflect"
	"strings"
	"testing"
)

func TestExactStableSortAndMissing(t *testing.T) {
	input := Data{Headers: []string{"value", "identity"}, Rows: [][]string{{"9007199254740993", "b"}, {"9007199254740992", "a"}, {"", "unknown"}, {"9007199254740993", "c"}}}
	got, err := input.Sort(SortKey{Column: 0, Kind: Number, Descending: true})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"9007199254740993", "b"}, {"9007199254740993", "c"}, {"9007199254740992", "a"}, {"", "unknown"}}
	if !reflect.DeepEqual(got.Rows, want) {
		t.Fatalf("%v", got.Rows)
	}
	got.Rows[0][0] = "changed"
	if input.Rows[0][0] != "9007199254740993" {
		t.Fatal("source changed")
	}
	if _, err := (Data{Headers: []string{"x"}, Rows: [][]string{{"unknown"}}}).Sort(SortKey{Kind: Number}); err == nil {
		t.Fatal("invented numeric value")
	}
}
func TestWidthsFitEveryColumn(t *testing.T) {
	budget, err := AvailableWidth(11906, 1077, 1077, 0, 400, 9746)
	if err != nil {
		t.Fatal(err)
	}
	widths, err := Widths(budget, []string{"4.23cm", "3.44cm", "3.44cm", "3.44cm", "3.44cm"})
	if err != nil {
		t.Fatal(err)
	}
	sum := 0
	for _, w := range widths {
		if w <= 0 {
			t.Fatal(widths)
		}
		sum += w
	}
	if sum != budget {
		t.Fatal(widths, budget)
	}
	exact, err := Widths(8000, []string{"1in", "20%", ""})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(exact, []int{1440, 1600, 4960}) {
		t.Fatal(exact)
	}
	for _, bad := range []string{"-2cm", "NaN", "1e99999pt", "0%"} {
		if _, err := Widths(8000, []string{bad}); err == nil {
			t.Fatal(bad)
		}
	}
}
func TestTransformsAndExplicitFormats(t *testing.T) {
	d := Data{Headers: []string{"key", "value"}, Rows: [][]string{{"one", "9007199254740993"}, {"two", "null"}}}
	selected, err := d.Select([]int{1, 0})
	if err != nil {
		t.Fatal(err)
	}
	if selected.Rows[0][0] != "9007199254740993" {
		t.Fatal(selected)
	}
	if _, err := d.Select([]int{0, 0}); err == nil {
		t.Fatal("duplicate admitted")
	}
	transposed, err := d.Transpose()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(transposed.Headers, []string{"key", "one", "two"}) {
		t.Fatal(transposed)
	}
	display, full, err := Shorten("分区_very_long_suffix_末尾", 12)
	if err != nil || len([]rune(display)) != 12 || !strings.HasSuffix(display, "_末尾") || full != "分区_very_long_suffix_末尾" {
		t.Fatal(display, full, err)
	}
	formatted, err := Value("9007199254740993", Format{Mode: "decimal", Digits: 2})
	if err != nil || formatted != "9007199254740993.00" {
		t.Fatal(formatted, err)
	}
	bytes, err := Value("1.5", Format{Mode: "bytes_mb", Digits: 1})
	if err != nil || bytes != "1.5 MiB" {
		t.Fatal(bytes, err)
	}
	if _, err := Value("unknown", Format{Mode: "decimal"}); err == nil {
		t.Fatal("unknown formatted as zero")
	}
}

func TestNumericExpansionIsBoundedBeforeAllocation(t *testing.T) {
	for _, text := range []string{"1e1000000000", "1e-1000000000", strings.Repeat("9", 4097)} {
		if _, err := Value(text, Format{Mode: "decimal"}); err == nil {
			t.Fatal("unbounded format accepted")
		}
		if _, err := Widths(8000, []string{text + "pt"}); err == nil {
			t.Fatal("unbounded width accepted")
		}
		if _, err := (Data{Headers: []string{"Number"}, Rows: [][]string{{text}}}).Sort(SortKey{Kind: Number}); err == nil {
			t.Fatal("unbounded sort accepted")
		}
	}
}
