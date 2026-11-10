package template

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/internal/opc"
	"github.com/marcuoli/go-document-docx/report"
)

func TestStrictLiteralTemplateAcrossRuns(t *testing.T) {
	d, _ := editing.New(report.Defaults())
	p, _ := d.Paragraph("Customer {{ NA", editing.ParagraphOptions{})
	if err := p.Append("ME }}", editing.TextStyle{}); err != nil {
		t.Fatal(err)
	}
	_, _ = d.Paragraph("Literal {{OTHER}}", editing.ParagraphOptions{})
	var b bytes.Buffer
	if err := d.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	original := bytes.Clone(b.Bytes())
	got, err := Render(context.Background(), b.Bytes(), map[string]string{"NAME": "<Example & Co> {{OTHER}}", "OTHER": "Exact 9007199254740993"}, Options{Document: report.Defaults()})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, b.Bytes()) {
		t.Fatal("input changed")
	}
	pkg, _ := opc.Read(context.Background(), got)
	body := string(pkg.Parts["word/document.xml"])
	if !strings.Contains(body, "&lt;Example &amp; Co&gt; {{OTHER}}") || !strings.Contains(body, "Exact 9007199254740993") {
		t.Fatal("literal substitution failed", body)
	}
	if _, err := Render(context.Background(), original, map[string]string{"NAME": "Present"}, Options{Document: report.Defaults()}); err == nil {
		t.Fatal("missing value accepted")
	}
}
func TestUnsupportedTemplateProgramsFail(t *testing.T) {
	for _, text := range []string{"{% for item in rows %}", "{{ NAME | safe }}", "{{broken"} {
		d, _ := editing.New(report.Defaults())
		_, _ = d.Paragraph(text, editing.ParagraphOptions{})
		var b bytes.Buffer
		_ = d.Write(context.Background(), &b)
		if _, err := Render(context.Background(), b.Bytes(), nil, Options{Document: report.Defaults()}); err == nil {
			t.Fatal(text)
		}
	}
}
