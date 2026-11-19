//go:build windows && integration

package word

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/publication"
	"github.com/marcuoli/go-document-docx/report"
	"github.com/marcuoli/go-document-docx/template"
)

func TestNativeWordFollowsComposedLiteralNavigation(t *testing.T) {
	if os.Getenv("DOCX_WORD_INTEGRATION") != "1" {
		t.Skip("explicit desktop Word integration opt-in required")
	}
	page := func(label string) []byte {
		d, err := document.New(report.Defaults())
		if err != nil {
			t.Fatal(err)
		}
		name := "scope"
		if label == "Main" {
			name = "Scope"
		}
		d.AddTable(document.Table{Headers: []string{"Section"}, Widths: []int{5000}, Rows: [][]document.Cell{{{Text: "Jump " + label, LinkTarget: name, ScreenTip: "Navigate " + label}}}})
		if label == "Main" {
			d.BookmarkNext("closing1_scope")
			d.Paragraph("Reserved candidate")
		}
		d.BookmarkNext(name)
		d.Paragraph(label + " destination")
		var b bytes.Buffer
		if err := d.Write(t.Context(), &b); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	composed, err := template.Compose(t.Context(), page("Main"), page("Imported"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "native-navigation.docx")
	if _, err := publication.Save(t.Context(), path, false, WithUpdatedFields(publication.FromBytes(composed), Options{TemporaryDirectory: dir})); err != nil {
		t.Fatal(err)
	}

	// Follow only synthetic internal links in a separately owned hidden instance.
	// The existing refresh step above exercises the actual publication decorator.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		t.Fatal(err)
	}
	defer ole.CoUninitialize()
	unknown, err := oleutil.CreateObject("Word.Application")
	if err != nil {
		t.Fatal(err)
	}
	defer unknown.Release()
	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Release()
	defer func() {
		if err := invoke(app, "Quit", 0); err != nil {
			t.Error(err)
		}
	}()
	for _, setting := range []struct {
		name  string
		value any
	}{{"Visible", false}, {"DisplayAlerts", int32(0)}} {
		if err := put(app, setting.name, setting.value); err != nil {
			t.Fatal(err)
		}
	}
	value := func(parent *ole.IDispatch, name string) any {
		t.Helper()
		v, err := get(parent, name)
		if err != nil {
			t.Fatal(err)
		}
		result := v.Value()
		if err := v.Clear(); err != nil {
			t.Fatal(err)
		}
		return result
	}
	security := value(app, "AutomationSecurity")
	if err := put(app, "AutomationSecurity", int32(3)); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := put(app, "AutomationSecurity", security); err != nil {
			t.Error(err)
		}
	}()
	options, err := dispatch(app, "Options", false)
	if err != nil {
		t.Fatal(err)
	}
	defer options.Release()
	oldLinks := value(options, "UpdateLinksAtOpen")
	if err := put(options, "UpdateLinksAtOpen", false); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := put(options, "UpdateLinksAtOpen", oldLinks); err != nil {
			t.Error(err)
		}
	}()
	documents, err := dispatch(app, "Documents", false)
	if err != nil {
		t.Fatal(err)
	}
	defer documents.Release()
	doc, err := dispatch(documents, "Open", true, path, false, true, false)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Release()
	defer func() {
		if err := invoke(doc, "Close", 0); err != nil {
			t.Error(err)
		}
	}()
	links, err := dispatch(doc, "Hyperlinks", false)
	if err != nil {
		t.Fatal(err)
	}
	defer links.Release()
	bookmarks, err := dispatch(doc, "Bookmarks", false)
	if err != nil {
		t.Fatal(err)
	}
	defer bookmarks.Release()
	expected := map[string]string{"Jump Main": "Main destination", "Jump Imported": "Imported destination"}
	if fmt.Sprint(value(links, "Count")) != "2" {
		t.Fatal("native Word lost navigation links")
	}
	for i := 1; i <= 2; i++ {
		link, err := dispatch(links, "Item", true, i)
		if err != nil {
			t.Fatal(err)
		}
		target := fmt.Sprint(value(link, "SubAddress"))
		linkRange, err := dispatch(link, "Range", false)
		if err != nil {
			link.Release()
			t.Fatal(err)
		}
		label := strings.Trim(fmt.Sprint(value(linkRange, "Text")), "\r\a ")
		linkRange.Release()
		bookmark, err := dispatch(bookmarks, "Item", true, target)
		if err != nil {
			link.Release()
			t.Fatal(err)
		}
		destination, err := dispatch(bookmark, "Range", false)
		bookmark.Release()
		if err != nil {
			link.Release()
			t.Fatal(err)
		}
		text := strings.Trim(fmt.Sprint(value(destination, "Text")), "\r\a ")
		start := fmt.Sprint(value(destination, "Start"))
		destination.Release()
		if expected[label] == "" || text != expected[label] {
			link.Release()
			t.Fatal("native link resolves to wrong detail", label, target, text)
		}
		if err := invoke(link, "Follow"); err != nil {
			link.Release()
			t.Fatal(err)
		}
		link.Release()
		selection, err := dispatch(app, "Selection", false)
		if err != nil {
			t.Fatal(err)
		}
		selected, err := dispatch(selection, "Range", false)
		selection.Release()
		if err != nil {
			t.Fatal(err)
		}
		actual := fmt.Sprint(value(selected, "Start"))
		selected.Release()
		if actual != start {
			t.Fatal("Word Follow did not navigate to exact bookmark", label, actual, start)
		}
		delete(expected, label)
	}
	if len(expected) != 0 {
		t.Fatal("native navigation missing", expected)
	}
	t.Log("Word", value(app, "Version"), "refreshed and followed both colliding-name composed links to their exact target paragraphs")
}
