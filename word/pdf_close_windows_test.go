//go:build windows

package word

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	ole "github.com/go-ole/go-ole"
)

func TestReconcilePDFClose(t *testing.T) {
	const header = "%PDF-1.7\n"
	const objects = "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [] /Count 0 >>\nendobj\n"
	table := fmt.Sprintf("xref\n0 3\n0000000000 65535 f \n%010d 00000 n \n%010d 00000 n \ntrailer\n<< /Size 3 /Root 1 0 R >>\n",
		len(header), len(header)+strings.Index(objects, "2 0 obj"))
	valid := header + objects + table + fmt.Sprintf("startxref\n%d\n%%%%EOF\n", len(header)+len(objects))
	hresult := uint32(rpcEDisconnected)
	disconnected := ole.NewError(uintptr(int32(hresult)))
	closeErr := fmt.Errorf("Word Close: %w", errors.Join(disconnected))
	other := errors.New("another Word error")

	for _, tc := range []struct {
		name    string
		pdf     string
		err     error
		wantErr bool
	}{
		{"complete", valid, closeErr, false},
		{"COM-exception-info", valid, ole.NewErrorWithSubError(rpcEDisconnected, "", ole.EXCEPINFO{}), false},
		{"missing", "", closeErr, true},
		{"truncated", strings.TrimSuffix(valid, "%%EOF\n"), closeErr, true},
		{"wrong-header", strings.Replace(valid, "%PDF-", "wrong", 1), closeErr, true},
		{"wrong-offset", strings.Replace(valid, fmt.Sprintf("startxref\n%d", len(header)+len(objects)), "startxref\n9", 1), closeErr, true},
		{"unrelated-error", valid, other, true},
		{"additional-cleanup-error", valid, fmt.Errorf("Word Close: %w", errors.Join(disconnected, other)), true},
		{"additional-COM-error", valid, ole.NewErrorWithSubError(rpcEDisconnected, "", other), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "report.pdf")
			if tc.pdf != "" {
				if err := os.WriteFile(path, []byte(tc.pdf), 0600); err != nil {
					t.Fatal(err)
				}
			}
			err := reconcilePDFClose(path, tc.err)
			if tc.wantErr && !errors.Is(err, tc.err) || !tc.wantErr && err != nil {
				t.Fatalf("reconcilePDFClose = %v, want original failure = %t", err, tc.wantErr)
			}
		})
	}
}

func TestRemovePDFStageWaitsForSharingViolation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "word-stage")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "input.docx")
	if err := os.WriteFile(path, []byte("staged input"), 0600); err != nil {
		t.Fatal(err)
	}
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ, syscall.FILE_SHARE_READ,
		nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); !errors.Is(err, syscall.Errno(32)) {
		_ = syscall.CloseHandle(handle)
		t.Fatalf("expected Windows sharing violation, got %v", err)
	}
	done := make(chan error, 1)
	time.AfterFunc(75*time.Millisecond, func() { done <- syscall.CloseHandle(handle) })
	err = removePDFStage(dir)
	if closeErr := <-done; err != nil || closeErr != nil {
		t.Fatal(err, closeErr)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("stage remains: %v", err)
	}
}
