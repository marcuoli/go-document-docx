package template

import (
	"fmt"
	"slices"
	"strings"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/internal/opc"
)

func existingStyleNames(root *etree.Element) map[string]bool {
	names := map[string]bool{}
	for _, style := range opc.Elements(root, opc.Word, "style") {
		for _, tag := range []string{"name", "aliases"} {
			if e := opc.Child(style, opc.Word, tag); e != nil {
				for _, name := range strings.Split(opc.Attr(e, opc.Word, "val"), ",") {
					names[strings.ToLower(strings.TrimSpace(name))] = true
				}
			}
		}
	}
	return names
}

func qualifyStyleNames(style *etree.Element, prefix string, used map[string]bool) {
	// Word resolves built-in style names and aliases as well as IDs. Imported
	// definitions must become custom styles, or Word can repair their collisions.
	setAttr(style, opc.Word, "customStyle", "1")
	unique := func(name string) string {
		base := prefix + strings.TrimSpace(name)
		candidate := base
		for n := 2; used[strings.ToLower(candidate)]; n++ {
			candidate = fmt.Sprintf("%s_%d", base, n)
		}
		used[strings.ToLower(candidate)] = true
		return candidate
	}
	name := opc.Ordered(style, "name", opc.StyleOrder)
	original := opc.Attr(name, opc.Word, "val")
	if original == "" {
		original = opc.Attr(style, opc.Word, "styleId")
	}
	setAttr(name, opc.Word, "val", unique(original))
	if aliases := opc.Child(style, opc.Word, "aliases"); aliases != nil {
		values := []string{}
		for _, alias := range strings.Split(opc.Attr(aliases, opc.Word, "val"), ",") {
			if strings.TrimSpace(alias) != "" {
				values = append(values, unique(alias))
			}
		}
		setAttr(aliases, opc.Word, "val", strings.Join(values, ","))
	}
}

func normalizeStyleRuns(root *etree.Element) {
	// CT_StyleRunProperties is ordered even though ordinary text-run properties
	// tolerate other sequences. Some existing templates append overrides; retain
	// their values while placing base Word properties before extension properties.
	rank := func(e *etree.Element) int {
		if e.NamespaceURI() == opc.Word {
			if n := slices.Index(opc.RunOrder, e.Tag); n >= 0 {
				return n
			}
		}
		return len(opc.RunOrder)
	}
	for _, pr := range opc.Elements(root, opc.Word, "rPr") {
		children := pr.ChildElements()
		slices.SortStableFunc(children, func(a, b *etree.Element) int { return rank(a) - rank(b) })
		for _, child := range children {
			pr.RemoveChild(child)
		}
		for _, child := range children {
			pr.AddChild(child)
		}
	}
}
