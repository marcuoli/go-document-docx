// Package navigation shares literal bookmark names and link presentation.
package navigation

import (
	"fmt"
	"strings"
)

// LinkColor is the default RGB used for styled navigation links.
const LinkColor = "315357"

// ValidateBookmarkName admits the editor's Word bookmark convention: 1..40 ASCII
// letters, digits or underscores, with a letter or underscore first.
func ValidateBookmarkName(name string) error {
	if len(name) == 0 || len(name) > 40 {
		return fmt.Errorf("bookmark length must be 1 to 40")
	}
	for i, r := range name {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9') {
			return fmt.Errorf("invalid bookmark name")
		}
	}
	return nil
}

// BookmarkKey compares names without ASCII case, matching Word's uniqueness
// rule for authored names. Keep the original spelling in serialized markup.
func BookmarkKey(name string) string {
	// Existing templates may contain non-ASCII names. Fold only the SDK's
	// authored ASCII alphabet, leaving every other byte and spelling unchanged.
	var folded []byte
	for i := range len(name) {
		if name[i] >= 'A' && name[i] <= 'Z' {
			if folded == nil {
				folded = []byte(name)
			}
			folded[i] = name[i] + ('a' - 'A')
		}
	}
	if folded == nil {
		return name
	}
	return string(folded)
}

const (
	cellTipBookmarkPrefix = "_DocxCellTip"
	cellTipMarkerPrefix   = "_DocxCellTipMark"
)

// CellScreenTipBookmark names a hidden literal ScreenTip destination.
func CellScreenTipBookmark(id int) string {
	return fmt.Sprintf("%s%d", cellTipBookmarkPrefix, id)
}

// CellScreenTipMarker names the hidden producer marker for the same range.
func CellScreenTipMarker(id int) string {
	return fmt.Sprintf("%s%d", cellTipMarkerPrefix, id)
}

// IsCellScreenTipBookmark recognizes a new literal ScreenTip destination name.
// The destination also needs a coextensive producer marker before PDF export.
func IsCellScreenTipBookmark(name string) bool {
	return numberedName(name, "_docxcelltip")
}

// IsCellScreenTipMarker recognizes a literal ScreenTip producer marker name.
func IsCellScreenTipMarker(name string) bool {
	return numberedName(name, "_docxcelltipmark")
}

func numberedName(name, lowerPrefix string) bool {
	suffix, ok := strings.CutPrefix(BookmarkKey(name), lowerPrefix)
	if !ok || len(suffix) == 0 || suffix[0] == '0' {
		return false
	}
	for _, digit := range suffix {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}
