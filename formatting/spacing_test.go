package formatting

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/internal/opc"
)

func TestSetParagraphSpaceBeforePreservesSelectedContentAndParts(t *testing.T) {
	pkg := textPackage(t, `<w:p><w:pPr><w:pStyle w:val="Heading1"/><w:keepNext/><w:keepLines/><w:pageBreakBefore/><w:spacing w:before="240" w:after="100" w:afterLines="200" w:afterAutospacing="1" w:line="264" w:lineRule="auto" w:beforeAutospacing="on"/><w:jc w:val="left"/><w:rPr><w:rFonts w:ascii="Arial"/></w:rPr></w:pPr><w:bookmarkStart w:id="7" w:name="retained"/><w:r><w:t>Exact &lt;value&gt; 9007199254740993</w:t></w:r><w:fldSimple w:instr="DATE"><w:r><w:t>Saved date</w:t></w:r></w:fldSimple><w:bookmarkEnd w:id="7"/></w:p>`+
		`<w:p><w:pPr><w:pStyle w:val="Heading2"/><w:spacing w:before="240"/></w:pPr><w:r><w:t>Unselected</w:t></w:r></w:p>`+
		`<w:tbl><w:tr><w:tc><w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Selected in table</w:t></w:r></w:p></w:tc></w:tr></w:tbl>`+
		`<w:p><w:r><w:t>No explicit style</w:t></w:r></w:p>`+
		`<w:p><w:r><w:drawing><w:txbxContent><w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Selected in textbox</w:t></w:r></w:p></w:txbxContent></w:drawing></w:r></w:p>`)
	pkg.Parts["word/media/retained.bin"] = []byte("unchanged artwork")
	input := packageBytes(t, pkg)
	snapshot := bytes.Clone(input)
	out, err := SetParagraphSpaceBefore(t.Context(), input, 720, "Heading1", "")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(input, snapshot) {
		t.Fatal("input mutated")
	}
	got, err := opc.Read(t.Context(), out)
	if err != nil {
		t.Fatal(err)
	}
	for name, original := range pkg.Parts {
		if name != "word/document.xml" && !bytes.Equal(original, got.Parts[name]) {
			t.Fatal("unselected part changed", name)
		}
	}
	beforeDoc, _ := pkg.XML("word/document.xml")
	afterDoc, _ := got.XML("word/document.xml")
	beforeP, afterP := opc.Elements(beforeDoc.Root(), opc.Word, "p"), opc.Elements(afterDoc.Root(), opc.Word, "p")
	var values []string
	for i, p := range afterP {
		pr := opc.Child(p, opc.Word, "pPr")
		spacing := opc.Child(pr, opc.Word, "spacing")
		values = append(values, opc.Attr(spacing, opc.Word, "before"))
		if paragraphText(p) != paragraphText(beforeP[i]) {
			t.Fatal("literal text changed", i)
		}
	}
	if !reflect.DeepEqual(values, []string{"720", "240", "720", "720", "720", "720"}) {
		t.Fatal("exact-style/table/textbox/unstyled selection changed", values)
	}
	first := opc.Child(afterP[0], opc.Word, "pPr")
	spacing := opc.Child(first, opc.Word, "spacing")
	for key, want := range map[string]string{"before": "720", "beforeAutospacing": "0", "after": "100", "afterLines": "200", "afterAutospacing": "1", "line": "264", "lineRule": "auto"} {
		if opc.Attr(spacing, opc.Word, key) != want {
			t.Fatal("unrelated spacing changed", key)
		}
	}
	for _, tag := range []string{"keepNext", "keepLines", "pageBreakBefore", "jc", "rPr"} {
		if opc.Child(first, opc.Word, tag) == nil {
			t.Fatal("pagination/font/alignment lost", tag)
		}
	}
	var order []string
	for _, e := range first.ChildElements() {
		order = append(order, e.Tag)
	}
	if !reflect.DeepEqual(order, []string{"pStyle", "keepNext", "keepLines", "pageBreakBefore", "spacing", "jc", "rPr"}) {
		t.Fatal("paragraph schema order changed", order)
	}
	fields := opc.Elements(afterDoc.Root(), opc.Word, "fldSimple")
	marks := opc.Elements(afterDoc.Root(), opc.Word, "bookmarkStart")
	if len(fields) != 1 || opc.Attr(fields[0], opc.Word, "instr") != "DATE" || len(marks) != 1 || opc.Attr(marks[0], opc.Word, "name") != "retained" {
		t.Fatal("field/bookmark changed")
	}
	again, err := SetParagraphSpaceBefore(t.Context(), out, 720, "Heading1", "")
	if err != nil || !bytes.Equal(again, out) {
		t.Fatal("repeated no-op changed bytes", err)
	}
	none, err := SetParagraphSpaceBefore(t.Context(), input, 600, "NoMatch")
	if err != nil || !bytes.Equal(none, input) {
		t.Fatal("no-match changed bytes", err)
	}
	none[0] ^= 1
	if !bytes.Equal(input, snapshot) {
		t.Fatal("no-op aliases caller input")
	}
}

func TestSetParagraphSpaceBeforeBoundsAndAliasedNamespace(t *testing.T) {
	for _, twips := range []int{0, 31680} {
		t.Run(map[bool]string{true: "zero", false: "maximum"}[twips == 0], func(t *testing.T) {
			pkg := textPackage(t, `<a:p xmlns:a="`+opc.Word+`" xmlns:x="urn:other"><a:pPr><a:pStyle a:val="Heading1"/><a:spacing a:before="240" a:beforeAutospacing="true" a:after="100" x:before="foreign" x:beforeLines="300"/><a:ind a:left="20"/></a:pPr><a:r><a:t>Literal</a:t></a:r></a:p>`)
			out, err := SetParagraphSpaceBefore(t.Context(), packageBytes(t, pkg), twips, "Heading1")
			if err != nil {
				t.Fatal(err)
			}
			result, _ := opc.Read(t.Context(), out)
			doc, _ := result.XML("word/document.xml")
			nodes := opc.Elements(doc.Root(), opc.Word, "spacing")
			if len(nodes) != 1 {
				t.Fatal("spacing duplicated")
			}
			node := nodes[0]
			want := "0"
			if twips != 0 {
				want = "31680"
			}
			if opc.Attr(node, opc.Word, "before") != want || opc.Attr(node, opc.Word, "beforeAutospacing") != "0" || opc.Attr(node, "urn:other", "before") != "foreign" {
				t.Fatal("namespace spacing override failed")
			}
			count := 0
			for _, attr := range node.Attr {
				if attr.Key == "before" && attr.NamespaceURI() == opc.Word {
					count++
				}
			}
			if count != 1 {
				t.Fatal("duplicate expanded attribute", count)
			}
		})
	}
}

func TestSetParagraphSpaceBeforeRejectsConflictsAndInvalidInput(t *testing.T) {
	valid := packageBytes(t, textPackage(t, `<w:p><w:r><w:t>Literal</w:t></w:r></w:p>`))
	for _, tc := range []struct {
		name  string
		twips int
		ids   []string
		input []byte
	}{
		{"negative", -1, []string{""}, valid}, {"too large", 31681, []string{""}, valid}, {"no IDs", 720, nil, valid}, {"many IDs", 720, make([]string, 257), valid}, {"long ID", 720, []string{strings.Repeat("x", 1025)}, valid}, {"XML ID", 720, []string{"bad\x00"}, valid}, {"invalid UTF8 ID", 720, []string{"\xff"}, valid}, {"bad ZIP", 720, []string{""}, []byte("bad")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if out, err := SetParagraphSpaceBefore(t.Context(), tc.input, tc.twips, tc.ids...); err == nil || out != nil {
				t.Fatal("invalid input returned document", err)
			}
		})
	}
	for _, kind := range []string{"paragraph", "style", "numbering"} {
		t.Run(kind, func(t *testing.T) {
			pkg := textPackage(t, `<w:p><w:r><w:t>Literal</w:t></w:r></w:p>`)
			switch kind {
			case "paragraph":
				pkg = textPackage(t, `<w:p><w:r><w:t>Earlier selected paragraph</w:t></w:r></w:p><w:p><w:pPr><w:spacing w:beforeLines="0"/></w:pPr><w:r><w:t>Literal</w:t></w:r></w:p>`)
			case "style":
				pkg.Parts["word/styles.xml"] = []byte(`<w:styles xmlns:w="` + opc.Word + `"><w:style w:styleId="Inherited"><w:pPr><w:spacing w:beforeLines="100"/></w:pPr></w:style></w:styles>`)
			case "numbering":
				pkg.Parts["word/numbering.xml"] = []byte(`<w:numbering xmlns:w="` + opc.Word + `"><w:abstractNum w:abstractNumId="1"><w:lvl w:ilvl="0"><w:pPr><w:spacing w:beforeLines="100"/></w:pPr></w:lvl></w:abstractNum></w:numbering>`)
			}
			input := packageBytes(t, pkg)
			snapshot := bytes.Clone(input)
			if out, err := SetParagraphSpaceBefore(t.Context(), input, 720, ""); err == nil || out != nil || !strings.Contains(err.Error(), "line-based") {
				t.Fatal("conflict not explicitly rejected", err)
			}
			if !bytes.Equal(input, snapshot) {
				t.Fatal("failure mutated input after an earlier selected paragraph")
			}
			if out, err := SetParagraphSpaceBefore(t.Context(), input, 720, "NoMatch"); err != nil || !bytes.Equal(out, input) {
				t.Fatal("unselected conflict changed no-op", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if out, err := SetParagraphSpaceBefore(ctx, valid, 720, ""); !errors.Is(err, context.Canceled) || out != nil {
		t.Fatal("cancellation returned output", err)
	}
	pkg := textPackage(t, "")
	pkg.Parts["word/document.xml"] = []byte(`<w:document xmlns:w="` + opc.Word + `"/>`)
	if out, err := SetParagraphSpaceBefore(t.Context(), packageBytes(t, pkg), 720, ""); err == nil || out != nil {
		t.Fatal("missing body accepted")
	}
	pkg.Parts["word/document.xml"] = []byte("<broken")
	if out, err := SetParagraphSpaceBefore(t.Context(), packageBytes(t, pkg), 720, ""); err == nil || out != nil {
		t.Fatal("malformed XML accepted")
	}
}

func TestSetParagraphSpaceBeforePreservesLocallyShadowedPrefixes(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"alias", `<a:p xmlns:a="` + opc.Word + `" xmlns:w="urn:foreign"><a:pPr><a:pStyle a:val="Heading1"/><a:spacing a:before="240" w:beforeAutospacing="foreign-auto"/></a:pPr><a:r><a:t>Literal</a:t></a:r></a:p>`},
		{"default", `<p xmlns="` + opc.Word + `" xmlns:w="urn:foreign" xmlns:spacingWord="urn:occupied"><pPr><pStyle xmlns:a="` + opc.Word + `" a:val="Heading1"/><spacing w:before="foreign-before" spacingWord:beforeAutospacing="foreign-auto"/></pPr><r><t>Literal</t></r></p>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := packageBytes(t, textPackage(t, tc.body))
			out, err := SetParagraphSpaceBefore(t.Context(), input, 720, "Heading1")
			if err != nil {
				t.Fatal(err)
			}
			pkg, _ := opc.Read(t.Context(), out)
			doc, _ := pkg.XML("word/document.xml")
			nodes := opc.Elements(doc.Root(), opc.Word, "spacing")
			if len(nodes) != 1 {
				t.Fatal("spacing missing")
			}
			node := nodes[0]
			if opc.Attr(node, opc.Word, "before") != "720" || opc.Attr(node, opc.Word, "beforeAutospacing") != "0" {
				t.Fatal("new attributes are not Word attributes")
			}
			if tc.name == "alias" && opc.Attr(node, "urn:foreign", "beforeAutospacing") != "foreign-auto" {
				t.Fatal("foreign attribute rebound")
			}
			if tc.name == "default" && (opc.Attr(node, "urn:foreign", "before") != "foreign-before" || opc.Attr(node, "urn:occupied", "beforeAutospacing") != "foreign-auto") {
				t.Fatal("foreign/occupied prefix attributes changed")
			}
			again, err := SetParagraphSpaceBefore(t.Context(), out, 720, "Heading1")
			if err != nil || !bytes.Equal(again, out) {
				t.Fatal("shadowed-prefix repeat not a no-op", err)
			}
		})
	}
}
