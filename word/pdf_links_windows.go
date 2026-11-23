//go:build windows

package word

import (
	"context"
	"errors"
	"fmt"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/marcuoli/go-document-docx/internal/navigation"
)

// Word exposes both OOXML hyperlinks and refreshed HYPERLINK fields through
// the same collection. Modify only the staged, read-only document in memory.
func removePDFScreenTipLinks(ctx context.Context, doc *ole.IDispatch) (returnedErr error) {
	links, err := dispatch(doc, "Hyperlinks", false)
	if err != nil {
		return err
	}
	defer links.Release()
	count, err := wordValue[int32](get(links, "Count"))
	if err != nil || count == 0 {
		return err
	}
	bookmarks, err := dispatch(doc, "Bookmarks", false)
	if err != nil {
		return err
	}
	defer bookmarks.Release()
	showHidden, err := wordValue[bool](get(bookmarks, "ShowHidden"))
	if err != nil {
		return err
	}
	if !showHidden {
		if err := put(bookmarks, "ShowHidden", true); err != nil {
			return err
		}
		defer func() { returnedErr = errors.Join(returnedErr, put(bookmarks, "ShowHidden", false)) }()
	}
	// Delete removes the link wrapper while retaining the text and its bookmark.
	// Walk backward because deleting an item renumbers the Word collection.
	for i := count; i >= 1; i-- {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := removePDFScreenTipLink(links, bookmarks, i); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func removePDFScreenTipLink(links, bookmarks *ole.IDispatch, i int32) error {
	link, err := dispatch(links, "Item", true, i)
	if err != nil {
		return err
	}
	defer link.Release()
	name, err := wordValue[string](get(link, "SubAddress"))
	if err != nil || !navigation.IsCellScreenTipBookmark(name) {
		return err
	}
	address, err := wordValue[string](get(link, "Address"))
	if err != nil || address != "" {
		return err
	}
	tip, err := wordValue[string](get(link, "ScreenTip"))
	if err != nil || tip == "" {
		return err
	}
	source, err := dispatch(link, "Range", false)
	if err != nil {
		return err
	}
	defer source.Release()
	const withinTable int32 = 12 // wdWithInTable
	inTable, err := wordValue[bool](get(source, "Information", withinTable))
	if err != nil || !inTable {
		return err
	}
	font, err := dispatch(source, "Font", false)
	if err != nil {
		return err
	}
	underline, err := wordValue[int32](get(font, "Underline"))
	font.Release()
	// Literal and editor navigation links are underlined even when they carry
	// a ScreenTip. Keep explicit links, including links to their own range.
	if err != nil || underline != 0 {
		return err
	}
	exists, err := wordValue[bool](oleutil.CallMethod(bookmarks, "Exists", name))
	if err != nil || !exists {
		return err
	}
	bookmark, err := dispatch(bookmarks, "Item", true, name)
	if err != nil {
		return err
	}
	defer bookmark.Release()
	destination, err := dispatch(bookmark, "Range", false)
	if err != nil {
		return err
	}
	defer destination.Release()
	sameRange, err := wordValue[bool](oleutil.CallMethod(source, "IsEqual", destination))
	if err != nil || !sameRange {
		return err
	}
	marked, err := hasScreenTipMarker(source)
	if err != nil || !marked {
		return err
	}
	return invoke(link, "Delete")
}

func hasScreenTipMarker(source *ole.IDispatch) (bool, error) {
	bookmarks, err := dispatch(source, "Bookmarks", false)
	if err != nil {
		return false, err
	}
	defer bookmarks.Release()
	if err := put(bookmarks, "ShowHidden", true); err != nil {
		return false, err
	}
	count, err := wordValue[int32](get(bookmarks, "Count"))
	if err != nil {
		return false, err
	}
	for i := int32(1); i <= count; i++ {
		bookmark, err := dispatch(bookmarks, "Item", true, i)
		if err != nil {
			return false, err
		}
		same, err := isScreenTipMarker(source, bookmark)
		bookmark.Release()
		if err != nil || same {
			return same, err
		}
	}
	return false, nil
}

func isScreenTipMarker(source, bookmark *ole.IDispatch) (bool, error) {
	name, err := wordValue[string](get(bookmark, "Name"))
	if err != nil || !navigation.IsCellScreenTipMarker(name) {
		return false, err
	}
	marker, err := dispatch(bookmark, "Range", false)
	if err != nil {
		return false, err
	}
	defer marker.Release()
	return wordValue[bool](oleutil.CallMethod(source, "IsEqual", marker))
}

func wordValue[T any](value *ole.VARIANT, err error) (result T, returnedErr error) {
	if value != nil {
		defer func() { returnedErr = errors.Join(returnedErr, value.Clear()) }()
	}
	if err != nil {
		return result, err
	}
	if value == nil {
		return result, errors.New("Word returned no property value")
	}
	result, ok := value.Value().(T)
	if !ok {
		return result, fmt.Errorf("Word returned unexpected property type %T", value.Value())
	}
	return result, nil
}
