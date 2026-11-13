package template

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/beevik/etree"
	word "github.com/marcuoli/go-document-docx/document"
	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/internal/opc"
	"github.com/marcuoli/go-document-docx/report"
)

func examplePage(t *testing.T, title string, c color.RGBA) []byte {
	t.Helper()
	d, err := editing.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Paragraph(title, editing.ParagraphOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Bookmark("scope"); err != nil {
		t.Fatal(err)
	}
	if err := p.Link("Scope", "scope", "Go to scope", true); err != nil {
		t.Fatal(err)
	}
	pixels := image.NewRGBA(image.Rect(0, 0, 4, 4))
	pixels.Set(0, 0, c)
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, pixels); err != nil {
		t.Fatal(err)
	}
	if err := p.Image(imageBytes.Bytes(), 720, 720); err != nil {
		t.Fatal(err)
	}
	if _, err := d.List("Action "+title, true, 0); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := d.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestClosingPageRemapsResourcesListsAndBookmarks(t *testing.T) {
	first := examplePage(t, "Main review", color.RGBA{R: 255, A: 255})
	last := examplePage(t, "Closing review", color.RGBA{B: 255, A: 255})
	original := bytes.Clone(first)
	result, err := Compose(context.Background(), first, last)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, original) {
		t.Fatal("main input changed")
	}
	p, err := opc.Read(context.Background(), result)
	if err != nil {
		t.Fatal(err)
	}
	xml, err := p.XML("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(p.Parts["word/document.xml"]), "Closing review") {
		t.Fatal("closing body missing")
	}
	names := map[string]bool{}
	for _, e := range opc.Elements(xml.Root(), opc.Word, "bookmarkStart") {
		name := opc.Attr(e, opc.Word, "name")
		if names[name] {
			t.Fatal("duplicate bookmark", name)
		}
		names[name] = true
	}
	if !names["scope"] || !names["Closing1_scope"] {
		t.Fatal(names)
	}
	rel, err := p.Relationships("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	images := 0
	for _, e := range rel.Root().ChildElements() {
		if strings.HasSuffix(e.SelectAttrValue("Type", ""), "/image") {
			images++
			target, err := opc.Resolve("word/document.xml", e.SelectAttrValue("Target", ""))
			if err != nil || len(p.Parts[target]) == 0 {
				t.Fatal(target, err)
			}
		}
	}
	if images != 2 {
		t.Fatal("image relationship collision", images)
	}
	number, err := p.XML("word/numbering.xml")
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, e := range opc.Elements(number.Root(), opc.Word, "num") {
		id := opc.Attr(e, opc.Word, "numId")
		if ids[id] {
			t.Fatal("duplicate numbering instance")
		}
		ids[id] = true
	}
	if len(ids) != 4 {
		t.Fatal("lists were not remapped", len(ids))
	}
	if _, err := Compose(context.Background(), result, last); err != nil {
		t.Fatal("second append failed", err)
	}
}
func TestTemplateImageAndUnsupportedClosingConstruct(t *testing.T) {
	d, _ := editing.New(report.Defaults())
	_, _ = d.Paragraph("{{IMAGE}}", editing.ParagraphOptions{})
	var source bytes.Buffer
	if err := d.Write(context.Background(), &source); err != nil {
		t.Fatal(err)
	}
	imageData := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var imageBytes bytes.Buffer
	_ = png.Encode(&imageBytes, imageData)
	rendered, err := Render(context.Background(), source.Bytes(), nil, Options{Document: report.Defaults(), Images: map[string]Image{"IMAGE": {Data: imageBytes.Bytes(), Width: 720}}})
	if err != nil {
		t.Fatal(err)
	}
	p, _ := opc.Read(context.Background(), rendered)
	if !strings.Contains(string(p.Parts["word/document.xml"]), "drawing") {
		t.Fatal("image absent")
	}
	bad, _ := opc.Read(context.Background(), source.Bytes())
	xml, _ := bad.XML("word/document.xml")
	body := opc.Child(xml.Root(), opc.Word, "body")
	body.CreateElement("w:footnoteReference").CreateAttr("w:id", "1")
	_ = bad.SetXML("word/document.xml", xml)
	var b bytes.Buffer
	_ = bad.Write(context.Background(), &b)
	if _, err := Compose(context.Background(), source.Bytes(), b.Bytes()); err == nil {
		t.Fatal("unsupported notes silently imported")
	}
}
func TestComposePaginatesImportedParagraphWithoutBreakRun(t *testing.T) {
	main := examplePage(t, "First page", color.RGBA{R: 255, A: 255})
	last := examplePage(t, "Last page", color.RGBA{B: 255, A: 255})
	before := bytes.Clone(last)
	out, err := Compose(t.Context(), main, last)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, last) {
		t.Fatal("closing input changed")
	}
	pkg, err := opc.Read(t.Context(), out)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := pkg.XML("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	for _, br := range opc.Elements(doc.Root(), opc.Word, "br") {
		if opc.Attr(br, opc.Word, "type") == "page" {
			t.Fatal("standalone break can spill onto an empty page")
		}
	}
	breaks := opc.Elements(doc.Root(), opc.Word, "pageBreakBefore")
	if len(breaks) != 1 || opc.Attr(breaks[0], opc.Word, "val") != "1" {
		t.Fatal("appended paragraph does not start a page")
	}
	paragraph := breaks[0].Parent().Parent()
	if !strings.Contains(paragraph.Text(), "Last page") {
		text := ""
		for _, node := range opc.Elements(paragraph, opc.Word, "t") {
			text += node.Text()
		}
		if !strings.Contains(text, "Last page") {
			t.Fatal("page break is detached from imported content")
		}
	}
}

func TestComposeTableFirstBodyUsesMinimalPaginatedBoundary(t *testing.T) {
	main := examplePage(t, "First", color.RGBA{R: 255, A: 255})
	d, err := word.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	d.AddTable(word.Table{Headers: []string{"Item"}, Widths: []int{4000}, Rows: [][]word.Cell{{{Text: "Closing table"}}}})
	var source bytes.Buffer
	if err := d.Write(t.Context(), &source); err != nil {
		t.Fatal(err)
	}
	before := bytes.Clone(source.Bytes())
	out, err := Compose(t.Context(), main, source.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, source.Bytes()) {
		t.Fatal("table-first input changed")
	}
	pkg, err := opc.Read(t.Context(), out)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := pkg.XML("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	breaks := opc.Elements(doc.Root(), opc.Word, "pageBreakBefore")
	if len(breaks) != 1 {
		t.Fatal("table-first body has no single page boundary")
	}
	props := breaks[0].Parent()
	spacing := opc.Child(props, opc.Word, "spacing")
	if spacing == nil || opc.Attr(spacing, opc.Word, "line") != "1" || opc.Attr(spacing, opc.Word, "lineRule") != "exact" {
		t.Fatal("table boundary consumes normal paragraph height")
	}
	if !strings.Contains(string(pkg.Parts["word/document.xml"]), "Closing table") {
		t.Fatal("imported table missing")
	}
}
func TestComposeEnablesAliasedPageBreakWithoutDuplicateValue(t *testing.T) {
	for _, disabled := range []string{"0", "false"} {
		t.Run(disabled, func(t *testing.T) {
			main := examplePage(t, "First", color.RGBA{R: 255, A: 255})
			last := examplePage(t, "Last", color.RGBA{B: 255, A: 255})
			pkg, err := opc.Read(t.Context(), last)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := pkg.XML("word/document.xml")
			if err != nil {
				t.Fatal(err)
			}
			body := opc.Child(doc.Root(), opc.Word, "body")
			props := opc.Properties(body.ChildElements()[0], "pPr")
			boundary := opc.Ordered(props, "pageBreakBefore", opc.ParagraphOrder)
			boundary.Space = "alias"
			boundary.CreateAttr("xmlns:alias", opc.Word)
			boundary.CreateAttr("alias:val", disabled)
			if err := pkg.SetXML("word/document.xml", doc); err != nil {
				t.Fatal(err)
			}
			var source bytes.Buffer
			if err := pkg.Write(t.Context(), &source); err != nil {
				t.Fatal(err)
			}
			out, err := Compose(t.Context(), main, source.Bytes())
			if err != nil {
				t.Fatal(err)
			}
			result, err := opc.Read(t.Context(), out)
			if err != nil {
				t.Fatal(err)
			}
			resultXML, err := result.XML("word/document.xml")
			if err != nil {
				t.Fatal(err)
			}
			breaks := opc.Elements(resultXML.Root(), opc.Word, "pageBreakBefore")
			if len(breaks) != 1 {
				t.Fatal("page boundary duplicated")
			}
			count := 0
			for _, a := range breaks[0].Attr {
				if a.Key == "val" && a.NamespaceURI() == opc.Word {
					count++
					if a.Value != "1" {
						t.Fatal("boundary remained disabled", a.Value)
					}
				}
			}
			if count != 1 {
				t.Fatal("expanded value attribute duplicated", count)
			}
		})
	}
}

func TestComposeOptionalTailSpacingPreservesContentAndDefault(t *testing.T) {
	main := examplePage(t, "Main content", color.RGBA{R: 255, A: 255})
	closing := examplePage(t, "Following content", color.RGBA{B: 255, A: 255})
	for _, kind := range []string{"spacing", "run", "bookmark", "page", "section", "revision", "property-comment", "extension"} {
		t.Run(kind, func(t *testing.T) {
			pkg, err := opc.Read(t.Context(), main)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := pkg.XML("word/document.xml")
			if err != nil {
				t.Fatal(err)
			}
			body := opc.Child(doc.Root(), opc.Word, "body")
			section := opc.Child(body, opc.Word, "sectPr")
			position := len(body.Child)
			if section != nil {
				position = section.Index()
			}
			tail := etree.NewElement("w:p")
			tail.CreateAttr("xmlns:w", opc.Word)
			tail.CreateAttr("w:rsidR", "00112233")
			pr := tail.CreateElement("w:pPr")
			pr.CreateElement("w:spacing").CreateAttr("w:after", "120")
			switch kind {
			case "run":
				tail.CreateElement("w:r").CreateElement("w:tab")
			case "bookmark":
				start := tail.CreateElement("w:bookmarkStart")
				start.CreateAttr("w:id", "100")
				start.CreateAttr("w:name", "tail_content")
				tail.CreateElement("w:bookmarkEnd").CreateAttr("w:id", "100")
			case "page":
				pr.CreateElement("w:pageBreakBefore")
			case "revision":
				pr.CreateElement("w:rPr").CreateElement("w:ins").CreateAttr("w:id", "7")
			case "property-comment":
				pr.AddChild(etree.NewComment("retain paragraph metadata"))
			case "extension":
				property := pr.CreateElement("w:rPr")
				property.CreateElement("ext:payload").CreateAttr("xmlns:ext", "urn:example:format-extension")
			case "section":
				pr.CreateElement("w:sectPr")
			}
			body.InsertChildAt(position, tail)
			if err := pkg.SetXML("word/document.xml", doc); err != nil {
				t.Fatal(err)
			}
			var input bytes.Buffer
			if err := pkg.Write(t.Context(), &input); err != nil {
				t.Fatal(err)
			}
			frozen := bytes.Clone(input.Bytes())
			for _, trim := range []bool{false, true} {
				var result []byte
				if trim {
					result, err = ComposeWithOptions(t.Context(), input.Bytes(), closing, ComposeOptions{TrimTrailingSpacing: true})
				} else {
					result, err = Compose(t.Context(), input.Bytes(), closing)
				}
				if err != nil {
					t.Fatal(err)
				}
				out, err := opc.Read(t.Context(), result)
				if err != nil {
					t.Fatal(err)
				}
				xml, err := out.XML("word/document.xml")
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, p := range opc.Elements(xml.Root(), opc.Word, "p") {
					if opc.Attr(p, opc.Word, "rsidR") == "00112233" {
						found = true
					}
				}
				want := !trim || kind != "spacing"
				if found != want {
					t.Fatalf("trim=%t kind=%s tail retained=%t, want=%t", trim, kind, found, want)
				}
				if !bytes.Equal(input.Bytes(), frozen) {
					t.Fatal("source input mutated")
				}
			}
		})
	}
}

type cancellingTailContext struct {
	context.Context
	remaining int
}

func (c *cancellingTailContext) Err() error {
	c.remaining--
	if c.remaining <= 0 {
		return context.Canceled
	}
	return nil
}
func TestTrimTailIsCancellableAndRetainsStructuralMetadata(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		body := etree.NewElement("w:body")
		body.CreateAttr("xmlns:w", opc.Word)
		content := body.CreateElement("w:p")
		content.CreateElement("w:r").CreateElement("w:t").SetText("Retained text")
		for i := 0; i < 1000; i++ {
			body.CreateElement("w:p")
			body.AddChild(etree.NewText("\n"))
		}
		section := body.CreateElement("w:sectPr")
		var ctx context.Context = t.Context()
		if cancel {
			ctx = &cancellingTailContext{Context: ctx, remaining: 3}
		}
		err := trimTrailingSpacing(ctx, body)
		if cancel {
			if err != context.Canceled || len(body.ChildElements()) != 1002 {
				t.Fatal("tail scan ignored cancellation or changed content", err)
			}
		} else {
			elements := body.ChildElements()
			if err != nil || len(elements) != 2 || elements[0] != content || elements[1] != section || section.Parent() != body {
				t.Fatal("tail/section ownership mismatch", err)
			}
		}
	}
}
