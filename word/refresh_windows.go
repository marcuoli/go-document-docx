//go:build windows

package word

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

var wordGate = make(chan struct{}, 1)

func refreshFile(ctx context.Context, path string) (returnedErr error) {
	select {
	case wordGate <- struct{}{}:
		defer func() { <-wordGate }()
	case <-ctx.Done():
		return ctx.Err()
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		// S_FALSE still requires a matching CoUninitialize.
		var failure *ole.OleError
		if !errors.As(err, &failure) || failure.Code() != 1 {
			return fmt.Errorf("initialize COM: %w", err)
		}
	}
	defer ole.CoUninitialize()
	if err := ctx.Err(); err != nil {
		return err
	}
	// CreateObject creates an owned application; GetActiveObject would attach to
	// a user's open documents and must not be used for this operation.
	unknown, err := oleutil.CreateObject("Word.Application")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer unknown.Release()
	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer app.Release()
	defer func() { returnedErr = errors.Join(returnedErr, invoke(app, "Quit", 0)) }()
	for _, setting := range []struct {
		name  string
		value any
	}{
		{"Visible", false}, {"DisplayAlerts", int32(0)},
	} {
		if err := put(app, setting.name, setting.value); err != nil {
			return err
		}
	}
	security, err := get(app, "AutomationSecurity")
	if err != nil {
		return err
	}
	oldSecurity := security.Value()
	if err := security.Clear(); err != nil {
		return err
	}
	if err := put(app, "AutomationSecurity", int32(3)); err != nil {
		return err
	}
	defer func() { returnedErr = errors.Join(returnedErr, put(app, "AutomationSecurity", oldSecurity)) }()
	options, err := dispatch(app, "Options", false)
	if err != nil {
		return err
	}
	defer options.Release()
	links, err := get(options, "UpdateLinksAtOpen")
	if err != nil {
		return err
	}
	oldLinks := links.Value()
	if err := links.Clear(); err != nil {
		return err
	}
	if err := put(options, "UpdateLinksAtOpen", false); err != nil {
		return err
	}
	defer func() { returnedErr = errors.Join(returnedErr, put(options, "UpdateLinksAtOpen", oldLinks)) }()
	documents, err := dispatch(app, "Documents", false)
	if err != nil {
		return err
	}
	defer documents.Release()
	if err := ctx.Err(); err != nil {
		return err
	}
	// ConfirmConversions, ReadOnly and AddToRecentFiles are all false. Only the
	// owned staging file is opened; no source or destination document is edited.
	doc, err := dispatch(documents, "Open", true, path, false, false, false)
	if err != nil {
		return err
	}
	defer doc.Release()
	defer func() { returnedErr = errors.Join(returnedErr, invoke(doc, "Close", 0)) }()
	if err := updateIndexes(ctx, doc, "Update"); err != nil {
		return err
	}
	if err := invoke(doc, "Repaginate"); err != nil {
		return err
	}
	if err := updateIndexes(ctx, doc, "UpdatePageNumbers"); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Updating arbitrary document fields could fetch external content or replace
	// caller-supplied template values. Refresh TOCs and caption indexes only.
	return invoke(doc, "Save")
}

// dispatch takes its own reference before clearing the returned VARIANT.
func dispatch(parent *ole.IDispatch, name string, method bool, args ...any) (*ole.IDispatch, error) {
	var value *ole.VARIANT
	var err error
	if method {
		value, err = oleutil.CallMethod(parent, name, args...)
	} else {
		value, err = oleutil.GetProperty(parent, name, args...)
	}
	if err != nil {
		if value != nil {
			err = errors.Join(err, value.Clear())
		}
		return nil, fmt.Errorf("Word %s: %w", name, err)
	}
	object := value.ToIDispatch()
	if object == nil {
		return nil, errors.Join(fmt.Errorf("Word %s returned no object", name), value.Clear())
	}
	object.AddRef()
	if err := value.Clear(); err != nil {
		object.Release()
		return nil, err
	}
	return object, nil
}
func invoke(parent *ole.IDispatch, name string, args ...any) error {
	value, err := oleutil.CallMethod(parent, name, args...)
	if value != nil {
		err = errors.Join(err, value.Clear())
	}
	if err != nil {
		return fmt.Errorf("Word %s: %w", name, err)
	}
	return nil
}
func put(parent *ole.IDispatch, name string, value any) error {
	result, err := oleutil.PutProperty(parent, name, value)
	if result != nil {
		err = errors.Join(err, result.Clear())
	}
	if err != nil {
		return fmt.Errorf("Word %s: %w", name, err)
	}
	return nil
}

// get releases a failed Invoke result even when COM returns it with an HRESULT error.
func get(parent *ole.IDispatch, name string) (*ole.VARIANT, error) {
	value, err := oleutil.GetProperty(parent, name)
	if err != nil {
		if value != nil {
			err = errors.Join(err, value.Clear())
		}
		return nil, fmt.Errorf("Word %s: %w", name, err)
	}
	return value, nil
}

// updateIndexes includes caption indexes, which also use TOC fields. Leaving
// their old cached PAGEREFs would preserve broken bookmarks after repagination.
func updateIndexes(ctx context.Context, doc *ole.IDispatch, method string) error {
	for _, name := range []string{"TablesOfContents", "TablesOfFigures"} {
		collection, err := dispatch(doc, name, false)
		if err != nil {
			return err
		}
		err = updateIndexCollection(ctx, collection, method)
		collection.Release()
		if err != nil {
			return err
		}
	}
	return ctx.Err()
}

func updateIndexCollection(ctx context.Context, collection *ole.IDispatch, method string) error {
	value, err := get(collection, "Count")
	if err != nil {
		return err
	}
	count, ok := value.Value().(int32)
	clearErr := value.Clear()
	if !ok || count < 0 {
		return errors.Join(errors.New("invalid Word index count"), clearErr)
	}
	if clearErr != nil {
		return clearErr
	}
	for i := int32(1); i <= count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		index, err := dispatch(collection, "Item", true, i)
		if err != nil {
			return err
		}
		err = invoke(index, method)
		index.Release()
		if err != nil {
			return err
		}
	}
	return ctx.Err()
}
