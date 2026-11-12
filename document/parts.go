package document

import (
	"fmt"
	"strings"
)

func (d *Document) fonts() string {
	var b strings.Builder
	b.WriteString(`<w:fonts xmlns:w="` + wordNS + `">`)
	seen := map[string]bool{}
	for _, name := range []string{d.options.Font, d.options.HeadingFont, d.options.SubheadingFont, tableStyleFor(d.options).HeaderFont} {
		if seen[name] {
			continue
		}
		seen[name] = true
		b.WriteString(`<w:font w:name="` + escape(name) + `">`)
		if d.options.FallbackFont != "" && d.options.FallbackFont != name {
			b.WriteString(`<w:altName w:val="` + escape(d.options.FallbackFont) + `"/>`)
		}
		b.WriteString(`</w:font>`)
	}
	b.WriteString(`</w:fonts>`)
	return b.String()
}

func (d *Document) styles() string {
	o := d.options
	base := fmt.Sprintf(`<w:styles xmlns:w="%s"><w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="%s" w:hAnsi="%s"/><w:color w:val="000000"/><w:sz w:val="%d"/></w:rPr></w:rPrDefault><w:pPrDefault><w:pPr><w:widowControl/><w:spacing w:after="100" w:line="264" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults>`, wordNS, escape(o.Font), escape(o.Font), o.BodySize)
	// Style run properties use a schema sequence: emphasis precedes color/size.
	style := func(id, font string, size int, extraP, extraR string) string {
		return fmt.Sprintf(`<w:style w:type="paragraph" w:styleId="%s"><w:name w:val="%s"/><w:basedOn w:val="Normal"/><w:pPr>%s</w:pPr><w:rPr><w:rFonts w:ascii="%s" w:hAnsi="%s"/>%s<w:color w:val="000000"/><w:sz w:val="%d"/></w:rPr></w:style>`, id, id, extraP, escape(font), escape(font), extraR, size)
	}
	base += `<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>`
	base += style("Title", o.HeadingFont, 60, `<w:keepNext/><w:spacing w:before="1440" w:after="360"/>`, `<w:b/>`)
	base += style("Subtitle", o.Font, 26, `<w:keepNext/><w:spacing w:after="360"/>`, "")
	for i, size := range []int{o.HeadingSize, o.SubheadingSize, o.BodySize} {
		font := o.HeadingFont
		if i > 0 {
			font = o.SubheadingFont
		}
		base += style(fmt.Sprintf("Heading%d", i+1), font, size, fmt.Sprintf(`<w:keepNext/><w:keepLines/><w:spacing w:before="240" w:after="100"/><w:outlineLvl w:val="%d"/>`, i), `<w:b/>`)
	}
	base += style("Lead", o.Font, o.BodySize, `<w:keepNext/><w:keepLines/>`, "")
	tableSpacing := `<w:spacing w:after="30" w:line="240" w:lineRule="auto"/>`
	headerSpacing := `<w:spacing w:after="30"/>`
	if o.TableStyle != nil {
		tableSpacing = fmt.Sprintf(`<w:spacing w:before="0" w:after="%d" w:line="240" w:lineRule="auto"/>`, o.TableStyle.ParagraphAfter)
		headerSpacing = tableSpacing
	}
	base += style("TableText", o.Font, o.TableSize, tableSpacing, "")
	base += style("TableHeader", tableStyleFor(o).HeaderFont, o.TableSize, `<w:keepNext/>`+headerSpacing, `<w:b/>`)
	base += style("TableAfter", o.Font, 12, `<w:spacing w:after="80" w:line="80" w:lineRule="exact"/>`, "")
	base += style("Footer", o.Font, 16, `<w:jc w:val="right"/>`, "")
	return base + `</w:styles>`
}

func (d *Document) header() string {
	return `<w:hdr xmlns:w="` + wordNS + `">` + paragraph("Footer", d.options.Header) + `</w:hdr>`
}
func (d *Document) footer() string {
	return `<w:ftr xmlns:w="` + wordNS + `"><w:p><w:pPr><w:pStyle w:val="Footer"/></w:pPr><w:r>` + textRun(d.options.Footer+"  "+d.options.PageLabel+" ") + `</w:r><w:fldSimple w:instr="PAGE"><w:r><w:t>1</w:t></w:r></w:fldSimple></w:p></w:ftr>`
}

const contentTypes = `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/><Override PartName="/word/settings.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.settings+xml"/><Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/><Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/><Override PartName="/word/fontTable.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.fontTable+xml"/></Types>`
const rootRelationships = `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rIdDocument" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`
const documentRelationships = `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rIdStyles" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/><Relationship Id="rIdSettings" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/settings" Target="settings.xml"/><Relationship Id="rIdHeader" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/><Relationship Id="rIdFooter" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/><Relationship Id="rIdFonts" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/fontTable" Target="fontTable.xml"/></Relationships>`
const settings = `<w:settings xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:compat><w:doNotExpandShiftReturn w:val="1"/></w:compat></w:settings>`
