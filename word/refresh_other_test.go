//go:build !windows

package word

import (
	"bytes"
	"context"
	"errors"
	"github.com/marcuoli/go-document-docx/publication"
	"testing"
)

func TestUnsupportedHostDoesNotWrite(t *testing.T) {
	var out bytes.Buffer
	err := WithUpdatedFields(publication.FromBytes([]byte("docx")), Options{TemporaryDirectory: t.TempDir()}).Write(context.Background(), &out)
	if !errors.Is(err, ErrUnavailable) || out.Len() != 0 {
		t.Fatal(err, out.Len())
	}
}
