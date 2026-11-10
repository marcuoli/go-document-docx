// Package fonts embeds caller-provided OpenType faces into a DOCX package.
// It does not discover, install or distribute fonts. Callers supply permitted
// font bytes and the family name used by document styles.
package fonts

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/internal/opc"
)

// Face contains a complete, caller-owned TTF/OTF face. Name is its Word family
// name. Bold and Italic select the embedded face slot; collections/subsets are
// unsupported. Embed copies bytes and checks OS/2 technical embedding flags.
type Face struct {
	Name         string
	Data         []byte
	Bold, Italic bool
}

// Embed returns a new package containing standards-based obfuscated font parts,
// font-table relationships and font keys. The input is untouched. Restricted or
// bitmap-only faces, missing OS/2 tables, duplicate face slots and oversized data
// fail before returning output. Technical flags do not replace caller permission.
func Embed(ctx context.Context, input []byte, faces []Face) ([]byte, error) {
	p, err := opc.Read(ctx, input)
	if err != nil {
		return nil, err
	}
	if len(faces) == 0 {
		return bytes.Clone(input), nil
	}
	initialRelations, err := p.Relationships("word/document.xml")
	if err != nil {
		return nil, err
	}
	fontPart := "word/fontTable.xml"
	fontRelated := false
	for _, relation := range initialRelations.Root().ChildElements() {
		if strings.HasSuffix(relation.SelectAttrValue("Type", ""), "/fontTable") {
			if fontRelated || relation.SelectAttrValue("TargetMode", "") == "External" {
				return nil, fmt.Errorf("ambiguous or external font table")
			}
			fontRelated = true
			fontPart, err = opc.Resolve("word/document.xml", relation.SelectAttrValue("Target", ""))
			if err != nil {
				return nil, err
			}
		}
	}
	if fontRelated {
		if _, exists := p.Parts[fontPart]; !exists {
			return nil, fmt.Errorf("related font table is missing")
		}
	}
	packageBytes := 0
	for _, part := range p.Parts {
		packageBytes += len(part)
	}
	table, err := p.XML(fontPart)
	if err != nil {
		if _, exists := p.Parts[fontPart]; exists {
			return nil, err
		}
		table = opc.NewXML("fonts")
	}
	opc.WordPrefix(table.Root())
	rels, err := p.Relationships(fontPart)
	if err != nil {
		return nil, err
	}
	if len(faces) > 32 {
		return nil, fmt.Errorf("at most 32 font faces are supported")
	}
	seen := map[string]bool{}
	for _, face := range faces {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if strings.TrimSpace(face.Name) == "" || len(face.Name) > 255 {
			return nil, fmt.Errorf("font family name is required")
		}
		if err := opc.Text(face.Name); err != nil {
			return nil, err
		}
		if len(face.Data) > opc.MaxPackage-packageBytes {
			return nil, fmt.Errorf("embedded fonts exceed package limit")
		}
		packageBytes += len(face.Data)
		if err := embeddingAllowed(face.Data); err != nil {
			return nil, err
		}
		slot := "embedRegular"
		if face.Bold && face.Italic {
			slot = "embedBoldItalic"
		} else if face.Bold {
			slot = "embedBold"
		} else if face.Italic {
			slot = "embedItalic"
		}
		key := face.Name + "/" + slot
		if seen[key] {
			return nil, fmt.Errorf("duplicate font face slot")
		}
		seen[key] = true
		var font *etree.Element
		for _, e := range table.Root().ChildElements() {
			if e.Tag == "font" && opc.Attr(e, opc.Word, "name") == face.Name {
				font = e
				break
			}
		}
		if font == nil {
			font = table.Root().CreateElement("w:font")
			font.CreateAttr("w:name", face.Name)
		}
		if opc.Child(font, opc.Word, slot) != nil {
			return nil, fmt.Errorf("font face slot already embedded")
		}
		guid := [16]byte{}
		if _, err := rand.Read(guid[:]); err != nil {
			return nil, err
		}
		encoded := bytes.Clone(face.Data)
		// ECMA-376 obfuscates the first 32 bytes using the GUID bytes in reverse
		// textual-byte order. The relationship points to this part, not to a raw TTF.
		for i := 0; i < 32; i++ {
			encoded[i] ^= guid[15-i%16]
		}
		name := ""
		for i := 1; ; i++ {
			name = fmt.Sprintf("word/fonts/font%d.odttf", i)
			if _, exists := p.Parts[name]; !exists {
				break
			}
		}
		p.Parts[name] = encoded
		rid := opc.AddRelation(rels, "font", opc.Relative(fontPart, name), "")
		entry := font.CreateElement("w:" + slot)
		entry.CreateAttr("r:id", rid)
		text := strings.ToUpper(hex.EncodeToString(guid[:]))
		entry.CreateAttr("w:fontKey", "{"+text[:8]+"-"+text[8:12]+"-"+text[12:16]+"-"+text[16:20]+"-"+text[20:]+"}")
		if err := p.ContentType(name, "application/vnd.openxmlformats-officedocument.obfuscatedFont"); err != nil {
			return nil, err
		}
	}
	if err := p.SetXML(fontPart, table); err != nil {
		return nil, err
	}
	if err := p.SetXML(opc.RelName(fontPart), rels); err != nil {
		return nil, err
	}
	if err := p.ContentType(fontPart, "application/vnd.openxmlformats-officedocument.wordprocessingml.fontTable+xml"); err != nil {
		return nil, err
	}
	documentRels, err := p.Relationships("word/document.xml")
	if err != nil {
		return nil, err
	}
	found := false
	for _, e := range documentRels.Root().ChildElements() {
		if strings.HasSuffix(e.SelectAttrValue("Type", ""), "/fontTable") {
			found = true
		}
	}
	if !found {
		opc.AddRelation(documentRels, "fontTable", opc.Relative("word/document.xml", fontPart), "")
	}
	if err := p.SetXML(opc.RelName("word/document.xml"), documentRels); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := p.Write(ctx, &out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
func embeddingAllowed(data []byte) error {
	if len(data) < 32 || len(data) > 32<<20 {
		return fmt.Errorf("font size is invalid")
	}
	signature := string(data[:4])
	if signature != "\x00\x01\x00\x00" && signature != "OTTO" && signature != "true" {
		return fmt.Errorf("font must be a single TTF/OTF face")
	}
	count := int(binary.BigEndian.Uint16(data[4:6]))
	if count < 1 || count > (len(data)-12)/16 {
		return fmt.Errorf("invalid font table directory")
	}
	for i := 0; i < count; i++ {
		entry := data[12+i*16 : 12+(i+1)*16]
		if string(entry[:4]) != "OS/2" {
			continue
		}
		offset, size := uint64(binary.BigEndian.Uint32(entry[8:12])), uint64(binary.BigEndian.Uint32(entry[12:16]))
		if size < 10 || offset > uint64(len(data)) || size > uint64(len(data))-offset {
			return fmt.Errorf("invalid OS/2 table")
		}
		flags := binary.BigEndian.Uint16(data[offset+8 : offset+10])
		if flags&0x0002 != 0 || flags&0x0200 != 0 {
			return fmt.Errorf("font flags forbid this embedding")
		}
		return nil
	}
	return fmt.Errorf("font has no OS/2 embedding flags")
}
