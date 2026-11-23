//go:build windows

package word

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	ole "github.com/go-ole/go-ole"
)

const rpcEDisconnected = 0x80010108

func reconcilePDFClose(path string, closeErr error) error {
	if !onlyCloseDisconnect(closeErr) {
		return closeErr
	}
	// Word can disconnect the document proxy after writing a complete PDF.
	// A failed or incomplete export must still reach the atomic publisher.
	if err := checkCompletedWordPDF(path); err != nil {
		return errors.Join(closeErr, err)
	}
	return nil
}

// A joined VARIANT cleanup error must survive even if Close disconnected.
func onlyCloseDisconnect(err error) bool {
	for err != nil {
		switch current := err.(type) {
		case *ole.OleError:
			// HRESULT is 32 bits even when Windows sign-extends the result.
			if uint32(current.Code()) != rpcEDisconnected {
				return false
			}
			// go-ole attaches EXCEPINFO to every failed IDispatch call,
			// even when it carries no separate Word error code.
			info, hasExceptionInfo := current.SubError().(ole.EXCEPINFO)
			return current.SubError() == nil || hasExceptionInfo && info.WCode() == 0 && info.SCODE() == 0
		case interface{ Unwrap() []error }:
			children := current.Unwrap()
			if len(children) != 1 {
				return false
			}
			err = children[0]
		default:
			err = errors.Unwrap(err)
		}
	}
	return false
}

func checkCompletedWordPDF(path string) (returnedErr error) {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("verify Word PDF after Close: %w", err)
	}
	defer func() { returnedErr = errors.Join(returnedErr, f.Close()) }()
	invalid := errors.New("verify Word PDF after Close: incomplete PDF")
	var header [5]byte
	if _, err := f.ReadAt(header[:], 0); err != nil || string(header[:]) != "%PDF-" {
		return invalid
	}
	info, err := f.Stat()
	if err != nil {
		return err
	}
	// Word writes a classic cross-reference table. Check its location and
	// terminal trailer using bounded reads, without loading the PDF into RAM.
	var buffer [1024]byte
	size := info.Size()
	start := max(0, size-int64(len(buffer)))
	tail := buffer[:size-start]
	if _, err := f.ReadAt(tail, start); err != nil {
		return invalid
	}
	marker := bytes.LastIndex(tail, []byte("startxref"))
	if marker < 0 {
		return invalid
	}
	fields := bytes.Fields(tail[marker:])
	if len(fields) != 3 || string(fields[0]) != "startxref" || string(fields[2]) != "%%EOF" {
		return invalid
	}
	offset, err := strconv.ParseInt(string(fields[1]), 10, 64)
	if err != nil || offset < int64(len(header)) || offset >= start+int64(marker) {
		return invalid
	}
	var xref [5]byte
	if _, err := f.ReadAt(xref[:], offset); err != nil || string(xref[:4]) != "xref" ||
		!bytes.ContainsRune([]byte("\r\n\t "), rune(xref[4])) {
		return invalid
	}
	return nil
}

func removePDFStage(path string) error {
	const (
		sharingViolation syscall.Errno = 32
		retryInterval                  = 25 * time.Millisecond
		maxWait                        = time.Second
	)
	deadline := time.Now().Add(maxWait)
	for {
		err := os.RemoveAll(path)
		// Word may return from Quit just before releasing its staged input.
		if !errors.Is(err, sharingViolation) || !time.Now().Before(deadline) {
			return err
		}
		time.Sleep(retryInterval)
	}
}
