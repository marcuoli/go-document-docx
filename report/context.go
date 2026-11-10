package report

import "strings"

// Component supplies a caller-authored context label and value. No application
// identifiers, translation, discovery or database naming rules are implied.
type Component struct{ Label, Value string }

// ScopedTitle joins only nonempty caller-supplied context components to a title.
// The caller supplies the separator and translated labels.
func ScopedTitle(title, separator string, components ...Component) string {
	values := []string{}
	for _, component := range components {
		if component.Value != "" {
			values = append(values, strings.TrimSpace(component.Label+" "+component.Value))
		}
	}
	if len(values) == 0 {
		return title
	}
	return title + separator + strings.Join(values, " ")
}
