package editing_test

import (
	"bytes"
	"context"
	"strconv"
	"testing"

	"github.com/marcuoli/go-document-docx/editing"
	"github.com/marcuoli/go-document-docx/internal/opc"
	"github.com/marcuoli/go-document-docx/report"
	"github.com/marcuoli/go-document-docx/tabledata"
	"github.com/mmonterroca/docxgo/v2/domain"
)

func TestSectionsKeepGeometryAndHeaderInheritance(t *testing.T) {
	d, _ := editing.New(report.Defaults())
	_, _ = d.Normal("First section")
	values := tabledata.Data{Headers: []string{"Key", "Value"}, Rows: [][]string{{"one", "9007199254740993"}}}
	if _, err := d.Table(values, editing.TableOptions{CapWidth: 31680}); err != nil {
		t.Fatal(err)
	}
	section, err := d.Native().AddSection()
	if err != nil {
		t.Fatal(err)
	}
	if err := section.SetPageSize(domain.PageSize{Width: 16838, Height: 11906}); err != nil {
		t.Fatal(err)
	}
	margins := section.Margins()
	margins.Left = 1440
	margins.Right = 1440
	if err := section.SetMargins(margins); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Table(values, editing.TableOptions{CapWidth: 31680}); err != nil {
		t.Fatal(err)
	}
	enumeration := section.(interface {
		HeadersAll() map[domain.HeaderType]domain.Header
		FootersAll() map[domain.FooterType]domain.Footer
	})
	beforeHeaders, beforeFooters := len(enumeration.HeadersAll()), len(enumeration.FootersAll())
	var b bytes.Buffer
	if err := d.Write(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	if len(enumeration.HeadersAll()) != beforeHeaders || len(enumeration.FootersAll()) != beforeFooters {
		t.Fatal("read-only validation created headers/footers")
	}
	p, _ := opc.Read(context.Background(), b.Bytes())
	xml, err := p.XML("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	tables := opc.Elements(xml.Root(), opc.Word, "tbl")
	if len(tables) != 2 {
		t.Fatal("tables lost")
	}
	for i, want := range []int{report.Defaults().PageWidth - 2*report.Defaults().Margin, 16838 - 2880} {
		preferred := opc.Child(opc.Child(tables[i], opc.Word, "tblPr"), opc.Word, "tblW")
		got, _ := strconv.Atoi(opc.Attr(preferred, opc.Word, "w"))
		if got != want {
			t.Fatal("wrong section budget", i, got, want)
		}
	}
}
