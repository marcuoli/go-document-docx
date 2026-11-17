package template

import (
	"context"
	"fmt"
)

// Count every occurrence before asking the synchronous upstream engine to expand
// values. Checking only distinct values permits a short template to exhaust memory.
func checkExpandedText(ctx context.Context, textBytes int64, counts map[string]int, syntax, values map[string]string) error {
	const limit int64 = 64 << 20
	// Remove all placeholder spellings first so shrinking keys can fund growing
	// ones regardless of map iteration order. Admission already bounds source bytes.
	for original, count := range counts {
		textBytes -= int64(len(original)) * int64(count)
	}
	if textBytes > limit {
		return fmt.Errorf("template output text exceeds 64 MiB")
	}
	for original, count := range counts {
		if err := ctx.Err(); err != nil {
			return err
		}
		size := int64(len(values[syntax[original]]))
		if size > (limit-textBytes)/int64(count) {
			return fmt.Errorf("template output text exceeds 64 MiB")
		}
		textBytes += size * int64(count)
	}
	return ctx.Err()
}
