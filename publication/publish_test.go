package publication_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/publication"
	"github.com/marcuoli/go-document-docx/report"
)

func TestAtomicPublicationPreservesPriorDocument(t *testing.T) {
	d, _ := document.New(report.Defaults())
	d.Title("Technical Review")
	path := filepath.Join(t.TempDir(), "report.docx")
	if installed, err := publication.Publish(context.Background(), path, false, d); err != nil || !installed {
		t.Fatalf("publication: %t %v", installed, err)
	}
	original, _ := os.ReadFile(path)
	if installed, err := publication.Publish(context.Background(), path, false, d); err == nil || installed {
		t.Fatal("existing document replaced without permission")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if installed, err := publication.Publish(ctx, path, true, d); !errors.Is(err, context.Canceled) || installed {
		t.Fatal("cancelled document installed")
	}
	bad, _ := document.New(report.Defaults())
	bad.Paragraph("invalid\x00")
	if installed, err := publication.Publish(context.Background(), path, true, bad); err == nil || installed {
		t.Fatal("invalid document installed")
	}
	var zero document.Document
	if installed, err := publication.Publish(context.Background(), path, true, &zero); err == nil || installed {
		t.Fatal("zero-value document installed")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(original) {
		t.Fatal("failed writes changed prior document")
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatal("staging files leaked")
	}
	if installed, err := publication.Publish(context.Background(), path, true, d); err != nil || !installed {
		t.Fatalf("explicit overwrite failed: %v", err)
	}
}
