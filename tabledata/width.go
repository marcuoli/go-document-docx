package tabledata

import (
	"fmt"
	"math/big"
	"slices"
	"strings"
)

// AvailableWidth returns the usable twips after margins, gutter and nonnegative
// table indentation, capped when capTwips is positive. It rejects unusable geometry.
func AvailableWidth(page, left, right, gutter, indent, capTwips int) (int, error) {
	if page <= 0 || page > 31680 || left < 0 || right < 0 || gutter < 0 || indent < 0 || left > page || right > page || gutter > page || indent > page {
		return 0, fmt.Errorf("invalid table page geometry")
	}
	n := page - left - right - gutter - indent
	if capTwips > 0 {
		n = min(n, capTwips)
	}
	if n < 1 {
		return 0, fmt.Errorf("no available table width")
	}
	return n, nil
}

// Widths resolves cm, mm, in, pt, percent and bare-point column widths to twips.
// Empty specifications share the remainder. Overcommitted grids scale every
// column proportionally; all columns stay positive and exactly fill available.
func Widths(available int, specs []string) ([]int, error) {
	if available < 1 || available > 31680 || len(specs) == 0 || len(specs) > 32 || len(specs) > available {
		return nil, fmt.Errorf("invalid width budget")
	}
	values := make([]int, len(specs))
	missing := []int{}
	total := 0
	for i, spec := range specs {
		spec = strings.ToLower(strings.TrimSpace(spec))
		if spec == "" {
			missing = append(missing, i)
			continue
		}
		n, err := width(spec, available)
		if err != nil {
			return nil, fmt.Errorf("column %d: %w", i, err)
		}
		if n < 1 || n > 1_000_000 {
			return nil, fmt.Errorf("column width out of range")
		}
		values[i] = n
		total += n
	}
	remainder := available - total
	if len(missing) > 0 {
		for j, i := range missing {
			values[i] = max(1, remainder/len(missing))
			if remainder > 0 && j < remainder%len(missing) {
				values[i]++
			}
		}
	} else if remainder >= 0 {
		values[len(values)-1] += remainder
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	if sum > available {
		type share struct {
			index     int
			remainder int64
		}
		shares := make([]share, len(values))
		used := len(values)
		for i, v := range values {
			numerator := int64(available-len(values)) * int64(v)
			values[i] = 1 + int(numerator/int64(sum))
			used += values[i] - 1
			shares[i] = share{i, numerator % int64(sum)}
		}
		slices.SortStableFunc(shares, func(a, b share) int { return int(b.remainder - a.remainder) })
		for _, s := range shares[:available-used] {
			values[s.index]++
		}
	}
	return values, nil
}

func width(spec string, available int) (int, error) {
	factor := big.NewRat(20, 1)
	for _, unit := range []struct {
		suffix string
		n, d   int64
	}{{"cm", 72000, 127}, {"mm", 7200, 127}, {"in", 1440, 1}, {"pt", 20, 1}, {"%", int64(available), 100}} {
		if strings.HasSuffix(spec, unit.suffix) {
			spec = strings.TrimSpace(strings.TrimSuffix(spec, unit.suffix))
			factor = big.NewRat(unit.n, unit.d)
			break
		}
	}
	n, err := number(spec)
	if err != nil || n.Sign() <= 0 {
		return 0, fmt.Errorf("invalid width")
	}
	n.Mul(n, factor)
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(n.Num(), n.Denom(), r)
	if new(big.Int).Lsh(r, 1).Cmp(n.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if !q.IsInt64() || q.Int64() > 1_000_000 {
		return 0, fmt.Errorf("width overflow")
	}
	return int(q.Int64()), nil
}
