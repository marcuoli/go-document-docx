package publication

import (
	"bytes"
	"context"
	"io"
)

// FromBytes takes an owned snapshot of an already serialized document. It does
// not validate its format. Template/font package results can be published without
// a second serialization; later caller changes cannot affect the saved artifact.
func FromBytes(data []byte) Source { return byteSource{data: bytes.Clone(data)} }

type byteSource struct{ data []byte }

func (s byteSource) Write(ctx context.Context, w io.Writer) error {
	for offset := 0; offset < len(s.data); {
		if err := ctx.Err(); err != nil {
			return err
		}
		end := min(offset+32<<10, len(s.data))
		n, err := w.Write(s.data[offset:end])
		if err != nil {
			return err
		}
		if n != end-offset {
			return io.ErrShortWrite
		}
		offset = end
	}
	return ctx.Err()
}
