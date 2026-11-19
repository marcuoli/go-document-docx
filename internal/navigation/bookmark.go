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
