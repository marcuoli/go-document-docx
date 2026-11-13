package formatting

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/internal/opc"
	"github.com/marcuoli/go-document-docx/report"
)

func TestAlignParagraphsPreservesTextPartsAndOtherStyles(t *testing.T) {
	pkg := textPackage(t,
		"<w:p><w:pPr><w:pStyle w:val=\"Normal\"/><w:keepNext/><w:jc w:val=\"left\"/></w:pPr><w:r><w:t>Exact &lt;value&gt; 9007199254740993</w:t></w:r></w:p>"+
			"<w:p><w:pPr><w:pStyle w:val=\"Lead\"/><w:keepLines/></w:pPr><w:r><w:t>Purpose</w:t></w:r></w:p>"+
			"<w:p><w:pPr><w:pStyle w:val=\"Heading1\"/></w:pPr><w:r><w:t>Heading</w:t></w:r></w:p>"+
			"<w:tbl><w:tr><w:tc><w:p><w:pPr><w:pStyle w:val=\"Normal\"/></w:pPr><w:r><w:t>Template prose</w:t></w:r></w:p></w:tc>"+
			"<w:tc><w:p><w:pPr><w:pStyle w:val=\"TableText\"/></w:pPr><w:r><w:t>Exact table cell</w:t></w:r></w:p></w:tc></w:tr></w:tbl>"+
			"<w:p><w:bookmarkStart w:id=\"7\" w:name=\"retained\"/><w:fldSimple w:instr=\"DATE\"><w:r><w:t>Saved date</w:t></w:r></w:fldSimple><w:bookmarkEnd w:id=\"7\"/></w:p>")
	pkg.Parts["word/media/retained.bin"] = []byte("unchanged artwork")
	input := packageBytes(t, pkg)
	snapshot := bytes.Clone(input)
	out, err := AlignParagraphs(t.Context(), input, "justify", "", "Normal", "Lead")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(input, snapshot) {
		t.Fatal("source mutated")
	}
	got, err := opc.Read(t.Context(), out)
	if err != nil {
		t.Fatal(err)
	}
	for name, original := range pkg.Parts {
		if name != "word/document.xml" && !bytes.Equal(original, got.Parts[name]) {
			t.Fatal("unselected package part changed", name)
		}
	}
	xml, err := got.XML("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	var alignments, texts []string
	for _, p := range opc.Elements(xml.Root(), opc.Word, "p") {
		a := ""
		if pr := opc.Child(p, opc.Word, "pPr"); pr != nil {
			if jc := opc.Child(pr, opc.Word, "jc"); jc != nil {
				a = opc.Attr(jc, opc.Word, "val")
			}
		}
		alignments = append(alignments, a)
		texts = append(texts, paragraphText(p))
	}
	if !reflect.DeepEqual(alignments, []string{"both", "both", "", "both", "", "both"}) ||
		!reflect.DeepEqual(texts, []string{"Exact <value> 9007199254740993", "Purpose", "Heading", "Template prose", "Exact table cell", "Saved date"}) {
		t.Fatal("selected alignment, literal data or protected styles changed", alignments, texts)
	}
	fields := opc.Elements(xml.Root(), opc.Word, "fldSimple")
	bookmarks := opc.Elements(xml.Root(), opc.Word, "bookmarkStart")
	if len(fields) != 1 || opc.Attr(fields[0], opc.Word, "instr") != "DATE" ||
		len(bookmarks) != 1 || opc.Attr(bookmarks[0], opc.Word, "name") != "retained" {
		t.Fatal("field instruction or bookmark changed")
	}
	again, err := AlignParagraphs(t.Context(), out, "justify", "", "Normal", "Lead")
	if err != nil || !bytes.Equal(again, out) {
		t.Fatal("repeated formatting changed an already aligned package", err)
	}
	untouched, err := AlignParagraphs(t.Context(), input, "justify", "Unselected")
	if err != nil || !bytes.Equal(untouched, input) {
		t.Fatal("no-match formatting changed input bytes", err)
	}
}

func TestAlignParagraphsRejectsInvalidOptionsAndCancellation(t *testing.T) {
	input := packageBytes(t, textPackage(t, "<w:p><w:r><w:t>Literal</w:t></w:r></w:p>"))
	for _, tc := range []struct {
		name, alignment string
		styles          []string
	}{
		{"empty alignment", "", []string{"Normal"}},
		{"invalid alignment", "both", []string{"Normal"}},
		{"XML injection", "justify\"/><w:r>", []string{"Normal"}},
		{"missing selection", "justify", nil},
		{"oversized selection", "justify", make([]string, 257)},
		{"oversized style", "justify", []string{strings.Repeat("x", 1025)}},
		{"invalid text", "justify", []string{"bad\x00style"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if out, err := AlignParagraphs(t.Context(), input, tc.alignment, tc.styles...); err == nil || out != nil {
				t.Fatal("invalid options returned a document", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if out, err := AlignParagraphs(ctx, input, "justify", ""); !errors.Is(err, context.Canceled) || out != nil {
		t.Fatal("cancelled formatting returned a document", err)
	}
}

func textPackage(t *testing.T, body string) *opc.Package {
	t.Helper()
	d, err := document.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := d.Write(t.Context(), &b); err != nil {
		t.Fatal(err)
	}
	p, err := opc.Read(t.Context(), b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	p.Parts["word/document.xml"] = []byte("<w:document xmlns:w=\"" + opc.Word + "\"><w:body>" + body + "<w:sectPr/></w:body></w:document>")
	return p
}

func packageBytes(t *testing.T, p *opc.Package) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := p.Write(t.Context(), &b); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func paragraphText(p *etree.Element) string {
	var b strings.Builder
	for _, text := range opc.Elements(p, opc.Word, "t") {
		b.WriteString(text.Text())
	}
	return b.String()
}

func TestAlignParagraphsUpdatesNamespaceAliasedAttributes(t *testing.T) {
	p := textPackage(t, "<a:p xmlns:a=\""+opc.Word+"\"><a:pPr><a:pStyle a:val=\"Normal\"/><a:jc a:val=\"left\"/></a:pPr><a:r><a:t>Retained text</a:t></a:r></a:p>")
	out, err := AlignParagraphs(t.Context(), packageBytes(t, p), "justify", "Normal")
	if err != nil {
		t.Fatal(err)
	}
	got, err := opc.Read(t.Context(), out)
	if err != nil {
		t.Fatal(err)
	}
	xml, err := got.XML("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	nodes := opc.Elements(xml.Root(), opc.Word, "jc")
	if len(nodes) != 1 || opc.Attr(nodes[0], opc.Word, "val") != "both" {
		t.Fatal("aliased alignment not updated")
	}
	count := 0
	for _, a := range nodes[0].Attr {
		if a.Key == "val" && a.NamespaceURI() == opc.Word {
			count++
		}
	}
	if count != 1 {
		t.Fatal("duplicate expanded attribute", count)
	}
}
