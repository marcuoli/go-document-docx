package tabledata

import (
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Format selects explicit display conversion. Empty Mode preserves literal text.
// Decimal uses exact rational arithmetic; DateTime requires explicit layouts.
// Bytes and Megabytes use binary units without a floating-point conversion.
type Format struct {
	Mode                      string
	Digits                    int
	InputLayout, OutputLayout string
}

// Value formats a display value; it never changes an original evidence store.
func Value(text string, format Format) (string, error) {
	if text == "" || format.Mode == "" {
		return text, nil
	}
	if format.Mode == "datetime" {
		if format.InputLayout == "" || format.OutputLayout == "" {
			return "", fmt.Errorf("date layouts are required")
		}
		t, err := time.Parse(format.InputLayout, text)
		if err != nil {
			return "", err
		}
		return t.Format(format.OutputLayout), nil
	}
	if format.Digits < 0 || format.Digits > 15 {
		return "", fmt.Errorf("digits must be 0 to 15")
	}
	n, err := number(strings.TrimSpace(text))
	if err != nil {
		return "", fmt.Errorf("not a numeric value")
	}
	switch format.Mode {
	case "decimal":
		return n.FloatString(format.Digits), nil
	case "bytes", "bytes_mb":
		if n.Sign() < 0 {
			return "", fmt.Errorf("negative byte count")
		}
		if format.Mode == "bytes_mb" {
			n.Mul(n, big.NewRat(1<<20, 1))
		}
		units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
		i := 0
		for n.Cmp(big.NewRat(1024, 1)) >= 0 && i < len(units)-1 {
			n.Quo(n, big.NewRat(1024, 1))
			i++
		}
		return n.FloatString(format.Digits) + " " + units[i], nil
	default:
		return "", fmt.Errorf("unknown format %q", format.Mode)
	}
}
