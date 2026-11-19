// Package navigation shares literal bookmark names and link presentation.
package navigation

import "fmt"

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
