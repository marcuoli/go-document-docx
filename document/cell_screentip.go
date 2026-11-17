package document

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Tooltip attributes need XML escaping without text-run breaks/tabs. Bound each
// escaped write instead of allocating a fully expanded attribute first (#33).
func writeScreenTipAttribute(ctx context.Context, b *strings.Builder, text string) error {
	output := screenTipAttributeWriter{ctx, b}
	for offset := 0; offset < len(text); {
		if err := ctx.Err(); err != nil {
			return err
		}
		end := min(offset+4096, len(text))
		for end < len(text) && !utf8.RuneStart(text[end]) {
			end--
		}
		if err := xml.EscapeText(output, []byte(text[offset:end])); err != nil {
			return err
		}
		offset = end
	}
	return ctx.Err()
}

type screenTipAttributeWriter struct {
	ctx    context.Context
	output *strings.Builder
}

func (w screenTipAttributeWriter) Write(value []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	if len(value) > (128<<20)-w.output.Len() {
		return 0, fmt.Errorf("document XML exceeds 128 MiB")
	}
	return w.output.Write(value)
}
