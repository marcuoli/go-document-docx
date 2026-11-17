package template

import (
	"bytes"
	"context"
	"errors"
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

func TestRenderRejectsExpandedTextBeforeReplacement(t *testing.T) {
	d, err := editing.New(report.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	for range 33 {
		if _, err := d.Paragraph("{{VALUE}}", editing.ParagraphOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	var input bytes.Buffer
	if err := d.Write(t.Context(), &input); err != nil {
		t.Fatal(err)
	}
	original := bytes.Clone(input.Bytes())
	out, err := Render(t.Context(), input.Bytes(), map[string]string{"VALUE": strings.Repeat("v", 2<<20)}, Options{Document: report.Defaults()})
	if err == nil || !strings.Contains(err.Error(), "template output text exceeds 64 MiB") || out != nil {
		t.Fatal("expanded text was not rejected by template preflight", err)
	}
	if !bytes.Equal(input.Bytes(), original) {
		t.Fatal("rejected template changed input")
	}
}

func TestExpandedTextBudgetCountsRepeatedSpellingsAndLiteralText(t *testing.T) {
	const limit = 64 << 20
	for _, tc := range []struct {
		name           string
		literal        int64
		counts         map[string]int
		syntax, values map[string]string
		wantErr        bool
	}{
		{"exact boundary", 0, map[string]int{"{{K}}": 4096}, map[string]string{"{{K}}": "K"}, map[string]string{"K": strings.Repeat("v", 16384)}, false},
		{"one byte per occurrence over", 0, map[string]int{"{{K}}": 4096}, map[string]string{"{{K}}": "K"}, map[string]string{"K": strings.Repeat("v", 16385)}, true},
		{"literal text counts", limit, map[string]int{"{{K}}": 1}, map[string]string{"{{K}}": "K"}, map[string]string{"K": "x"}, true},
		{"different spellings share value", limit - 3, map[string]int{"{{K}}": 1, "{{ K }}": 1}, map[string]string{"{{K}}": "K", "{{ K }}": "K"}, map[string]string{"K": "xx"}, true},
		{"shrinking replacements", limit, map[string]int{"{{LONG}}": 1}, map[string]string{"{{LONG}}": "LONG"}, map[string]string{"LONG": ""}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			size := tc.literal
			for original, count := range tc.counts {
				size += int64(len(original) * count)
			}
			err := checkExpandedText(t.Context(), size, tc.counts, tc.syntax, tc.values)
			if (err != nil) != tc.wantErr {
				t.Fatal("expanded text budget", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := checkExpandedText(ctx, 0, nil, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation identity lost", err)
	}
}
