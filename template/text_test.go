package template

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/internal/opc"
)

func textPackage(t *testing.T, body string) *opc.Package {
	t.Helper()
	p := blankPackage(t, "Original")
	p.Parts["word/document.xml"] = []byte("<w:document xmlns:w=\"" + opc.Word + "\" xmlns:r=\"" + opc.Office + "\" xmlns:v=\"urn:schemas-microsoft-com:vml\"><w:body>" + body + "<w:sectPr/></w:body></w:document>")
	return p
}
func TestRenderTextPreservesSplitRunArtworkAndFields(t *testing.T) {
	p := textPackage(t, "<w:p><w:r><w:fldChar w:fldCharType=\"begin\"/></w:r><w:r><w:instrText> PAGE </w:instrText></w:r><w:r><w:fldChar w:fldCharType=\"end\"/></w:r><w:r><w:rPr><w:b/></w:rPr><w:t>Å {{ FIR</w:t></w:r><w:r><w:t>ST }} | {{SECOND}} | {{FIRST}}</w:t></w:r><w:r><w:pict><v:shape id=\"artwork\"/></w:pict></w:r></w:p>")
	p.Parts["word/media/unchanged.bin"] = []byte("owned artwork")
	original := packageBytes(t, p)
	snapshot := bytes.Clone(original)
	out, err := RenderText(context.Background(), original, map[string]string{"FIRST": "<& {{SECOND}} é", "SECOND": "二"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, snapshot) {
		t.Fatal("input changed")
	}
	output, _ := opc.Read(context.Background(), out)
	xml, err := output.XML("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	text := paragraphText(opc.Elements(xml.Root(), opc.Word, "p")[0])
	if text != "Å <& {{SECOND}} é | 二 | <& {{SECOND}} é" {
		t.Fatal(text)
	}
	if len(opc.Elements(xml.Root(), opc.Word, "pict")) != 1 || len(opc.Elements(xml.Root(), opc.Word, "fldChar")) != 2 || len(opc.Elements(xml.Root(), opc.Word, "b")) != 1 {
		t.Fatal("artwork/fields/formatting lost")
	}
	for name, data := range p.Parts {
		if name != "word/document.xml" && !bytes.Equal(data, output.Parts[name]) {
			t.Fatal("untouched part changed", name)
		}
	}
}
func TestRenderTextHeadersAndNestedParagraphs(t *testing.T) {
	p := textPackage(t, "<w:p><w:r><w:t>{{OUTER}}</w:t></w:r><w:r><w:drawing><w:txbxContent><w:p><w:r><w:t>{{INNER}}</w:t></w:r></w:p></w:txbxContent></w:drawing></w:r></w:p>")
	p.Parts["word/header1.xml"] = []byte("<w:hdr xmlns:w=\"" + opc.Word + "\"><w:p><w:r><w:t>{{OUTER}}</w:t></w:r></w:p></w:hdr>")
	out, err := RenderText(context.Background(), packageBytes(t, p), map[string]string{"OUTER": "Outside", "INNER": "Inside"})
	if err != nil {
		t.Fatal(err)
	}
	output, _ := opc.Read(context.Background(), out)
	xml, _ := output.XML("word/document.xml")
	paragraphs := opc.Elements(xml.Root(), opc.Word, "p")
	if paragraphText(paragraphs[0]) != "Outside" || paragraphText(paragraphs[1]) != "Inside" {
		t.Fatal("nested paragraph duplicated")
	}
	header, _ := output.XML("word/header1.xml")
	if paragraphText(opc.Elements(header.Root(), opc.Word, "p")[0]) != "Outside" {
		t.Fatal("header unchanged")
	}
}
func TestRenderTextRejectsUnsupportedLocationsAndControls(t *testing.T) {
	for _, body := range []string{
		"<w:p><w:r><w:t>{{MISSING}}</w:t></w:r></w:p>",
		"<w:p><w:r><w:delInstrText>{{KEY}}</w:delInstrText></w:r></w:p>",
		"<w:p><w:r><w:t>{{KEY|filter}}</w:t></w:r></w:p>",
		"<w:p><w:r><w:t>{% if KEY %}</w:t></w:r></w:p>",
		"<w:p><w:r><w:t>{{KEY</w:t></w:r><w:r><w:br/><w:t>}}</w:t></w:r></w:p>",
		"<w:p><w:r><w:instrText>{</w:instrText></w:r><w:r><w:instrText>{KEY}}</w:instrText></w:r></w:p>",
		"<w:p><w:fldSimple w:instr=\"PAGE\"><w:r><w:t>{{KEY}}</w:t></w:r></w:fldSimple></w:p>",
		"<w:p w:rsidR=\"{{KEY}}\"><w:r><w:t>Text</w:t></w:r></w:p>",
		"<w:p><w:ins><w:r><w:t>{{KEY}}</w:t></w:r></w:ins></w:p>",
		"<w:ins><w:p><w:r><w:t>{{KEY}}</w:t></w:r></w:p></w:ins>",
	} {
		_, err := RenderText(context.Background(), packageBytes(t, textPackage(t, body)), map[string]string{"KEY": "value"})
		if err == nil {
			t.Fatal("unsupported template accepted", body)
		}
	}
}
func TestRenderTextBoundsAndCancellation(t *testing.T) {
	p := textPackage(t, "<w:p><w:r><w:t>{{KEY}}</w:t></w:r></w:p>")
	input := packageBytes(t, p)
	for _, values := range []map[string]string{{"KEY": "bad\x01"}, {"KEY": "line\nbreak"}, {"bad-key": "x"}} {
		if _, err := RenderText(context.Background(), input, values); err == nil {
			t.Fatal("invalid value admitted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RenderText(ctx, input, nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	many := textPackage(t, "<w:p><w:r><w:t>"+strings.Repeat("{{KEY}}", 4097)+"</w:t></w:r></w:p>")
	if _, err := RenderText(context.Background(), packageBytes(t, many), map[string]string{"KEY": "x"}); err == nil {
		t.Fatal("placeholder bound ignored")
	}
}

func TestRenderTextRejectsComplexFieldResultsAcrossParagraphs(t *testing.T) {
	for _, middle := range []string{
		"<w:r><w:t>{{KEY}}</w:t></w:r>",
		"</w:p><w:p><w:r><w:t>{{KEY}}</w:t></w:r>",
	} {
		body := "<w:p><w:r><w:fldChar w:fldCharType=\"begin\"/></w:r><w:r><w:instrText>PAGE</w:instrText></w:r><w:r><w:fldChar w:fldCharType=\"separate\"/></w:r>" + middle + "<w:r><w:fldChar w:fldCharType=\"end\"/></w:r></w:p>"
		if _, err := RenderText(context.Background(), packageBytes(t, textPackage(t, body)), map[string]string{"KEY": "value"}); err == nil {
			t.Fatal("complex field result accepted")
		}
	}
}

func TestRenderTextRejectsInlineObjectBoundaries(t *testing.T) {
	for _, object := range []string{"<w:cr/>", "<w:ptab/>", "<w:sym w:font=\"Symbol\" w:char=\"F020\"/>"} {
		body := "<w:p><w:r><w:t>{{KE</w:t>" + object + "<w:t>Y}}</w:t></w:r></w:p>"
		if _, err := RenderText(context.Background(), packageBytes(t, textPackage(t, body)), map[string]string{"KEY": "value"}); err == nil {
			t.Fatal("inline boundary admitted", object)
		}
	}
}

func TestRenderTextExplicitSimpleFieldCache(t *testing.T) {
	body := "<w:p><w:fldSimple w:instr=\" STYLEREF Title \\\\* MERGEFORMAT \">" + "<w:r><w:t>{{KEY}}</w:t></w:r></w:fldSimple></w:p>"
	p := textPackage(t, body)
	input := packageBytes(t, p)
	if _, err := RenderText(context.Background(), input, map[string]string{"KEY": "Review"}); err == nil {
		t.Fatal("implicit field cache accepted")
	}
	out, err := RenderTextWithOptions(context.Background(), input, map[string]string{"KEY": "Review"}, TextOptions{SimpleFieldResults: []string{"STYLEREF Title \\\\* MERGEFORMAT"}})
	if err != nil {
		t.Fatal(err)
	}
	output, _ := opc.Read(context.Background(), out)
	xml, _ := output.XML("word/document.xml")
	field := opc.Elements(xml.Root(), opc.Word, "fldSimple")[0]
	if opc.Attr(field, opc.Word, "instr") != " STYLEREF Title \\\\* MERGEFORMAT " || paragraphText(opc.Elements(xml.Root(), opc.Word, "p")[0]) != "Review" {
		t.Fatal("instruction/cache changed")
	}
}
