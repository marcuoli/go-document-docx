package tabledata

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// Bound both spelling and exponent before math/big expands scientific notation.
// Document display/sorting must not allocate an unbounded integer from a short value.
func number(text string) (*big.Rat, error) {
	if len(text) == 0 || len(text) > 4096 {
		return nil, fmt.Errorf("numeric value exceeds supported precision")
	}
	if index := strings.LastIndexAny(text, "eEpP"); index >= 0 {
		exponent, err := strconv.ParseInt(text[index+1:], 10, 32)
		if err != nil || exponent < -4096 || exponent > 4096 {
			return nil, fmt.Errorf("numeric exponent exceeds supported precision")
		}
	}
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, fmt.Errorf("not a numeric value")
	}
	return value, nil
}
