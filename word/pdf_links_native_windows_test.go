//go:build windows && integration

package word

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/publication"
	"github.com/marcuoli/go-document-docx/report"
)

func TestNativeWordPDFRemovesOnlyAutomaticScreenTipLinks(t *testing.T) {
	if os.Getenv("DOCX_WORD_INTEGRATION") != "1" {
		t.Skip("explicit desktop Word integration opt-in required")
	}
	doc, err := document.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	doc.BookmarkNext("actual_heading")
	doc.Heading("Actual heading", 1)
	const explicitHidden = "_DocxCellTip41"
	const explicitLegacy = "DocxCellTip42"
	doc.AddTable(document.Table{Headers: []string{"Links"}, Widths: []int{5000}, Rows: [][]document.Cell{
		{{Text: "Automatic tip", ScreenTip: "Expected automatic tip"}},
		{{Text: "Named tip", Bookmark: "named_tip", ScreenTip: "Named user bookmark"}},
		{{Text: "Explicit self", Bookmark: explicitHidden, LinkTarget: explicitHidden, ScreenTip: "Styled authored self"}},
		{{Text: "Legacy name", Bookmark: explicitLegacy, LinkTarget: explicitLegacy}},
		{{Text: "Heading jump", LinkTarget: "actual_heading", ScreenTip: "Jump to the heading"}},
		{{Text: "Other jump", LinkTarget: explicitHidden, ScreenTip: "Jump to the authored link"}},
		{{Text: "Named anchor jump", LinkTarget: "named_tip"}},
	}})
	dir := t.TempDir()
	for _, refresh := range []bool{false, true} {
		var source publication.Source = doc
		path := filepath.Join(dir, "raw.docx")
		if refresh {
			source = WithUpdatedFields(doc, Options{TemporaryDirectory: dir})
			path = filepath.Join(dir, "refreshed.docx")
		}
		if _, err := publication.Save(t.Context(), path, false, source); err != nil {
			t.Fatal(err)
		}
		err := withDocument(t.Context(), path, true, func(doc *ole.IDispatch) error {
			links, err := dispatch(doc, "Hyperlinks", false)
			if err != nil {
				return err
			}
			defer links.Release()
			bookmarks, err := dispatch(doc, "Bookmarks", false)
			if err != nil {
				return err
			}
			defer bookmarks.Release()
			var generated []string
			visit := func(count int32, visited func(string, string)) {
				t.Helper()
				for i := int32(1); i <= count; i++ {
					link, err := dispatch(links, "Item", true, i)
					if err != nil {
						t.Fatal(err)
					}
					name, nameErr := wordValue[string](get(link, "SubAddress"))
					tip, tipErr := wordValue[string](get(link, "ScreenTip"))
					link.Release()
					if nameErr != nil || tipErr != nil {
						t.Fatal(nameErr, tipErr)
					}
					visited(name, tip)
				}
			}
			count, err := wordValue[int32](get(links, "Count"))
			if err != nil || count != 7 {
				t.Fatal(count, err)
			}
			visit(count, func(name, tip string) {
				if tip == "Expected automatic tip" || tip == "Named user bookmark" {
					generated = append(generated, name)
				}
			})
			if len(generated) != 2 || generated[0] == generated[1] {
				t.Fatalf("Word did not retain both independent tooltip destinations: %q", generated)
			}
			for _, name := range generated {
				if !strings.HasPrefix(name, "_DocxCellTip") || name == explicitHidden {
					t.Fatalf("Word did not retain the hidden ScreenTip destination: %q", name)
				}
			}
			if err := put(bookmarks, "ShowHidden", false); err != nil {
				return err
			}
			visible, err := wordValue[int32](get(bookmarks, "Count"))
			if err != nil || visible != 3 {
				t.Fatalf("default visible bookmark list contains generated anchors: %d %v", visible, err)
			}
			if err := removePDFScreenTipLinks(t.Context(), doc); err != nil {
				return err
			}
			showHidden, err := wordValue[bool](get(bookmarks, "ShowHidden"))
			if err != nil || showHidden {
				t.Fatal(showHidden, err)
			}
			count, err = wordValue[int32](get(links, "Count"))
			if err != nil || count != 5 {
				t.Fatal(count, err)
			}
			want := map[string]int{explicitHidden: 2, explicitLegacy: 1, "actual_heading": 1, "named_tip": 1}
			visit(count, func(name, _ string) { want[name]-- })
			for name, n := range want {
				if n != 0 {
					t.Fatalf("wrong preserved link count for %q: %d", name, n)
				}
			}
			if err := put(bookmarks, "ShowHidden", true); err != nil {
				return err
			}
			for _, name := range append(generated, "named_tip") {
				exists, err := wordValue[bool](oleutil.CallMethod(bookmarks, "Exists", name))
				if err != nil || !exists {
					t.Fatal("both automatic and authored bookmarks must survive", name, exists, err)
				}
			}
			named, err := dispatch(bookmarks, "Item", true, "named_tip")
			if err != nil {
				return err
			}
			defer named.Release()
			target, err := dispatch(named, "Range", false)
			if err != nil {
				return err
			}
			defer target.Release()
			text, err := wordValue[string](get(target, "Text"))
			if err != nil || strings.TrimRight(text, "\r\a") != "Named tip" {
				t.Fatal("authored destination changed", text, err)
			}
			return nil
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
}
