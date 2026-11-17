//go:build windows && integration

package word

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/publication"
	"github.com/marcuoli/go-document-docx/report"
)

func TestNativeWordRetainsCellScreenTips(t *testing.T) {
	if os.Getenv("DOCX_WORD_INTEGRATION") != "1" {
		t.Skip("explicit desktop Word integration opt-in required")
	}
	d, err := document.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	tip := "Present <&> \"quoted\" ação"
	d.AddTable(document.Table{Headers: []string{"State"}, Widths: []int{5000}, FontSize: 14, Rows: [][]document.Cell{
		{{Text: "✔", TextColor: "000000", Bold: true, Alignment: "right", ScreenTip: tip}},
		{{Runs: []document.CellRun{{Text: "✘", TextColor: "B3261E", Italic: true}, {Text: "\n✔ ⚠️", TextColor: "000000"}}, ScreenTip: tip}},
	}})
	dir := t.TempDir()
	path := filepath.Join(dir, "native-screentip.docx")
	if _, err := publication.Save(t.Context(), path, false, WithUpdatedFields(d, Options{TemporaryDirectory: dir})); err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var main []byte
	for _, f := range z.File {
		if f.Name == "word/document.xml" {
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			main, e = io.ReadAll(r)
			r.Close()
			if e != nil {
				t.Fatal(e)
			}
		}
	}
	if len(main) == 0 {
		t.Fatal("missing Word-saved document")
	}
	decoder := xml.NewDecoder(bytes.NewReader(main))
	links := 0
	active := false
	italic := false
	red, black := false, false
	var text strings.Builder
	for {
		token, e := decoder.Token()
		if errors.Is(e, io.EOF) {
			break
		}
		if e != nil {
			t.Fatal(e)
		}
		switch node := token.(type) {
		case xml.StartElement:
			attrs := map[string]string{}
			for _, a := range node.Attr {
				attrs[a.Name.Local] = a.Value
			}
			if node.Name.Local == "hyperlink" {
				if attrs["tooltip"] != tip || attrs["anchor"] == "" || attrs["id"] != "" {
					t.Fatal("Word changed tooltip/internal target", attrs)
				}
				active = true
				links++
			}
			if active {
				switch node.Name.Local {
				case "t":
					var v string
					if e := decoder.DecodeElement(&v, &node); e != nil {
						t.Fatal(e)
					}
					text.WriteString(v)
				case "i":
					italic = attrs["val"] != "0" && attrs["val"] != "false"
				case "rStyle":
					if attrs["val"] == "Hyperlink" || attrs["val"] == "FollowedHyperlink" {
						t.Fatal("Word introduced link styling")
					}
				case "u":
					if attrs["val"] != "none" {
						t.Fatal("Word introduced underline", attrs)
					}
				case "color":
					red = red || attrs["val"] == "B3261E"
					black = black || attrs["val"] == "000000"
				}
			}
		case xml.EndElement:
			if node.Name.Local == "hyperlink" {
				active = false
			}
		}
	}
	if links != 2 || !italic || !red || !black || text.String() != "✔✘✔ ⚠️" {
		t.Fatal("Word changed literal linked appearance/content", links, italic, red, black, text.String())
	}
	t.Log("Word retained both native tooltip strings/internal anchors and original literal glyphs, italic/color properties without forced link styling; actual hover/UI acceptance remains separate")
}
