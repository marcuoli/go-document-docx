package template

import (
	"bytes"
	"context"
	"fmt"
	"path"

	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/marcuoli/go-document-docx/internal/opc"
)

// Compose appends a caller-supplied closing-page body after a page break, retaining
// the main document's page geometry, headers, footers, theme and core properties.
// It isolates imported style IDs, names and aliases, and remaps numbering,
// bookmarks and transitive image/link relationships. Style run-property order is
// normalized in the output without changing property values or source bytes.
// Closing documents with section breaks, notes/comments, altChunks, tracked changes
// or reference fields are rejected explicitly; this is not an arbitrary-file merger.
// Neither input is changed and no external relationship is fetched.
func Compose(ctx context.Context, main, closing []byte) ([]byte, error) {
	dest, err := opc.Read(ctx, main)
	if err != nil {
		return nil, err
	}
	source, err := opc.Read(ctx, closing)
	if err != nil {
		return nil, err
	}
	master, err := dest.XML("word/document.xml")
	if err != nil {
		return nil, err
	}
	extra, err := source.XML("word/document.xml")
	if err != nil {
		return nil, err
	}
	body := opc.Child(master.Root(), opc.Word, "body")
	other := opc.Child(extra.Root(), opc.Word, "body")
	if body == nil || other == nil {
		return nil, fmt.Errorf("missing document body")
	}
	for _, tag := range []string{"footnoteReference", "endnoteReference", "commentReference", "commentRangeStart", "commentRangeEnd", "altChunk", "ins", "del", "moveFrom", "moveTo"} {
		if len(opc.Elements(other, opc.Word, tag)) > 0 {
			return nil, fmt.Errorf("unsupported closing-page construct %s", tag)
		}
	}
	sections := opc.Elements(other, opc.Word, "sectPr")
	for _, s := range sections {
		if s.Parent() != other {
			return nil, fmt.Errorf("closing-page section breaks are unsupported")
		}
	}
	if err := checkComplexFields(other); err != nil {
		return nil, err
	}
	for _, e := range opc.Elements(other, opc.Word, "instrText") {
		if referenceField(e.Text()) {
			return nil, fmt.Errorf("closing-page reference fields are unsupported")
		}
	}
	for _, e := range opc.Elements(other, opc.Word, "fldSimple") {
		if referenceField(opc.Attr(e, opc.Word, "instr")) {
			return nil, fmt.Errorf("closing-page reference fields are unsupported")
		}
	}
	stylesPart, err := relatedPart(dest, "styles", "word/styles.xml")
	if err != nil {
		return nil, err
	}
	prefix := ""
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("Closing%d_", i)
		used := bytes.Contains(dest.Parts[stylesPart], []byte(candidate))
		for name := range dest.Parts {
			if strings.HasPrefix(name, fmt.Sprintf("word/closing%d/", i)) {
				used = true
			}
		}
		if !used {
			prefix = candidate
			break
		}
	}
	merger := &composer{ctx: ctx, source: source, dest: dest, prefix: prefix, imported: map[string]string{}, styleIDs: map[string]string{}, numIDs: map[string]string{}, abstractIDs: map[string]string{}}
	if err := merger.shared(); err != nil {
		return nil, err
	}
	relations, err := dest.Relationships("word/document.xml")
	if err != nil {
		return nil, err
	}
	srcRels, err := source.Relationships("word/document.xml")
	if err != nil {
		return nil, err
	}
	ids, err := merger.relationships("word/document.xml", "word/document.xml", srcRels, relations, true)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	maxID := 0
	for _, e := range opc.Elements(master.Root(), opc.Word, "bookmarkStart") {
		names[opc.Attr(e, opc.Word, "name")] = true
	}
	for _, tag := range []string{"bookmarkStart", "bookmarkEnd"} {
		for _, e := range opc.Elements(master.Root(), opc.Word, tag) {
			n, err := strconv.Atoi(opc.Attr(e, opc.Word, "id"))
			if err != nil {
				return nil, err
			}
			maxID = max(maxID, n)
		}
	}
	bookmarkIDs, bookmarkNames := map[string]string{}, map[string]string{}
	for _, e := range opc.Elements(other, opc.Word, "bookmarkStart") {
		oldID, oldName := opc.Attr(e, opc.Word, "id"), opc.Attr(e, opc.Word, "name")
		if oldID == "" || oldName == "" || bookmarkIDs[oldID] != "" {
			return nil, fmt.Errorf("invalid closing bookmark")
		}
		maxID++
		bookmarkIDs[oldID] = strconv.Itoa(maxID)
		name := oldName
		if names[name] {
			name = prefix + oldName
		}
		if len(name) > 40 {
			name = fmt.Sprintf("%sbookmark_%d", prefix, maxID)
		}
		if names[name] {
			return nil, fmt.Errorf("closing bookmark collision")
		}
		names[name] = true
		bookmarkNames[oldName] = name
	}
	drawingID := 0
	for _, e := range all(master.Root()) {
		if e.Tag == "docPr" {
			if i, err := strconv.Atoi(e.SelectAttrValue("id", "0")); err == nil {
				drawingID = max(drawingID, i)
			}
		}
	}
	content := []*etree.Element{}
	for _, child := range other.ChildElements() {
		if child.NamespaceURI() == opc.Word && child.Tag == "sectPr" {
			continue
		}
		cloned := inherited(child, extra.Root())
		merger.rewrite(cloned)
		for _, e := range all(cloned) {
			for i, a := range e.Attr {
				if a.NamespaceURI() == opc.Office && (a.Key == "id" || a.Key == "embed" || a.Key == "link") {
					target, ok := ids[a.Value]
					if !ok {
						return nil, fmt.Errorf("unmapped closing relationship")
					}
					e.Attr[i].Value = target
				}
			}
			if e.NamespaceURI() == opc.Word {
				if e.Tag == "bookmarkStart" || e.Tag == "bookmarkEnd" {
					old := opc.Attr(e, opc.Word, "id")
					replacement, ok := bookmarkIDs[old]
					if !ok {
						return nil, fmt.Errorf("unpaired closing bookmark")
					}
					setAttr(e, opc.Word, "id", replacement)
				}
				if e.Tag == "bookmarkStart" {
					setAttr(e, opc.Word, "name", bookmarkNames[opc.Attr(e, opc.Word, "name")])
				}
				if e.Tag == "hyperlink" {
					if name := opc.Attr(e, opc.Word, "anchor"); name != "" {
						if replacement := bookmarkNames[name]; replacement != "" {
							setAttr(e, opc.Word, "anchor", replacement)
						}
					}
				}
				if e.Tag == "p" {
					pr := opc.Child(e, opc.Word, "pPr")
					if pr == nil {
						pr = etree.NewElement("w:pPr")
						pr.CreateAttr("xmlns:w", opc.Word)
						e.InsertChildAt(0, pr)
					}
					if opc.Child(pr, opc.Word, "pStyle") == nil && merger.defaultStyle != "" {
						style := etree.NewElement("w:pStyle")
						style.CreateAttr("xmlns:w", opc.Word)
						style.CreateAttr("w:val", merger.defaultStyle)
						pr.InsertChildAt(0, style)
					}
				}
			}
			if e.Tag == "docPr" || e.Tag == "cNvPr" {
				drawingID++
				e.CreateAttr("id", strconv.Itoa(drawingID))
			}
		}
		content = append(content, cloned)
	}
	// A body-level sectPr must remain last. The page break and imported body precede it.
	position := len(body.Child)
	if section := opc.Child(body, opc.Word, "sectPr"); section != nil {
		position = section.Index()
	}
	page := etree.NewElement("w:p")
	page.CreateAttr("xmlns:w", opc.Word)
	page.CreateElement("w:r").CreateElement("w:br").CreateAttr("w:type", "page")
	body.InsertChildAt(position, page)
	position++
	for _, child := range content {
		body.InsertChildAt(position, child)
		position++
	}
	if err := dest.SetXML("word/document.xml", master); err != nil {
		return nil, err
	}
	if err := dest.SetXML(opc.RelName("word/document.xml"), relations); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := dest.Write(ctx, &output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

type composer struct {
	ctx                           context.Context
	source, dest                  *opc.Package
	prefix                        string
	imported                      map[string]string
	styleIDs, numIDs, abstractIDs map[string]string
	defaultStyle                  string
}

func referenceField(text string) bool {
	fields := strings.Fields(strings.ToUpper(strings.TrimSpace(text)))
	if len(fields) == 0 {
		return false
	}
	switch fields[0] {
	case "REF", "PAGEREF", "NOTEREF", "STYLEREF", "TOC":
		return true
	}
	return false
}

// Materialize the nearest namespace declarations from every ancestor, including
// body/container-local aliases. Copying only document-root declarations can
// leave imported elements or attributes with unbound or rebound prefixes.
func inherited(e, root *etree.Element) *etree.Element {
	out := e.Copy()
	for ancestor := e; ancestor != nil; ancestor = ancestor.Parent() {
		for _, a := range ancestor.Attr {
			if a.Space == "xmlns" || a.Space == "" && a.Key == "xmlns" {
				key := a.Key
				if a.Space != "" {
					key = a.Space + ":" + a.Key
				}
				if out.SelectAttr(key) == nil {
					out.CreateAttr(key, a.Value)
				}
			}
		}
	}
	return out
}
func all(root *etree.Element) []*etree.Element {
	out := []*etree.Element{}
	var walk func(*etree.Element)
	walk = func(e *etree.Element) {
		out = append(out, e)
		for _, c := range e.ChildElements() {
			walk(c)
		}
	}
	walk(root)
	return out
}
func setAttr(e *etree.Element, namespace, key, value string) {
	for i, a := range e.Attr {
		if a.Key == key && a.NamespaceURI() == namespace {
			e.Attr[i].Value = value
			return
		}
	}
	e.CreateAttr("w:"+key, value)
}
func (c *composer) rewrite(root *etree.Element) {
	for _, e := range all(root) {
		if e.NamespaceURI() != opc.Word {
			continue
		}
		val := opc.Attr(e, opc.Word, "val")
		switch e.Tag {
		case "pStyle", "rStyle", "tblStyle", "basedOn", "next", "link", "styleLink", "numStyleLink":
			if replacement := c.styleIDs[val]; replacement != "" {
				setAttr(e, opc.Word, "val", replacement)
			}
		case "numId":
			if replacement := c.numIDs[val]; replacement != "" {
				setAttr(e, opc.Word, "val", replacement)
			}
		case "abstractNumId":
			if replacement := c.abstractIDs[val]; replacement != "" {
				setAttr(e, opc.Word, "val", replacement)
			}
		}
	}
}
func relative(part, target string) string { return opc.Relative(part, target) }
func (c *composer) relationships(oldPart, newPart string, source, dest *etree.Document, body bool) (map[string]string, error) {
	ids := map[string]string{}
	for _, r := range source.Root().ChildElements() {
		kind := r.SelectAttrValue("Type", "")
		suffix := kind[strings.LastIndex(kind, "/")+1:]
		if body {
			switch suffix {
			// Note references are rejected above. Copying their unused parts would
			// create multiple singleton note relationships in the combined package.
			case "styles", "numbering", "fontTable", "settings", "theme", "webSettings", "header", "footer", "footnotes", "endnotes":
				continue
			}
		}
		target, mode := r.SelectAttrValue("Target", ""), r.SelectAttrValue("TargetMode", "")
		if mode != "External" {
			resolved, err := opc.Resolve(oldPart, target)
			if err != nil {
				return nil, err
			}
			imported, err := c.part(resolved)
			if err != nil {
				return nil, err
			}
			target = relative(newPart, imported)
		}
		id := opc.AddRelation(dest, suffix, target, mode)
		dest.Root().ChildElements()[len(dest.Root().ChildElements())-1].CreateAttr("Type", kind)
		ids[r.SelectAttrValue("Id", "")] = id
	}
	return ids, nil
}
func (c *composer) part(name string) (string, error) {
	if err := c.ctx.Err(); err != nil {
		return "", err
	}
	if existing := c.imported[name]; existing != "" {
		return existing, nil
	}
	data, ok := c.source.Parts[name]
	if !ok {
		return "", fmt.Errorf("closing relationship target is missing")
	}
	folder := strings.ToLower(strings.TrimSuffix(c.prefix, "_"))
	newName := "word/" + folder + "/" + name
	c.imported[name] = newName
	c.dest.Parts[newName] = bytes.Clone(data)
	kind, err := contentType(c.source, name)
	if err != nil {
		return "", err
	}
	if err := c.dest.ContentType(newName, kind); err != nil {
		return "", err
	}
	if _, has := c.source.Parts[opc.RelName(name)]; has {
		rel, err := c.source.Relationships(name)
		if err != nil {
			return "", err
		}
		rewritten := opc.RelationshipDocument()
		for _, r := range rel.Root().ChildElements() {
			target, mode := r.SelectAttrValue("Target", ""), r.SelectAttrValue("TargetMode", "")
			if mode != "External" {
				resolved, err := opc.Resolve(name, target)
				if err != nil {
					return "", err
				}
				imported, err := c.part(resolved)
				if err != nil {
					return "", err
				}
				target = relative(newName, imported)
			}
			cloned := inherited(r, rel.Root())
			cloned.CreateAttr("Target", target)
			rewritten.Root().AddChild(cloned)
		}
		if err := c.dest.SetXML(opc.RelName(newName), rewritten); err != nil {
			return "", err
		}
	}
	return newName, nil
}
func contentType(p *opc.Package, name string) (string, error) {
	d, err := p.XML("[Content_Types].xml")
	if err != nil {
		return "", err
	}
	for _, e := range d.Root().ChildElements() {
		if e.Tag == "Override" && e.SelectAttrValue("PartName", "") == "/"+name {
			return e.SelectAttrValue("ContentType", ""), nil
		}
	}
	extension := strings.TrimPrefix(path.Ext(name), ".")
	for _, e := range d.Root().ChildElements() {
		if e.Tag == "Default" && e.SelectAttrValue("Extension", "") == extension {
			return e.SelectAttrValue("ContentType", ""), nil
		}
	}
	return "", fmt.Errorf("closing part has no content type")
}

// Field instructions may be fragmented across runs; inspect complete instructions
// before bookmark remapping. A partial "RE"/"F scope" must not evade rejection.
func checkComplexFields(root *etree.Element) error {
	type field struct {
		instruction strings.Builder
		code        bool
	}
	stack := []*field{}
	for _, element := range all(root) {
		if element.NamespaceURI() != opc.Word {
			continue
		}
		if element.Tag == "fldChar" {
			switch opc.Attr(element, opc.Word, "fldCharType") {
			case "begin":
				stack = append(stack, &field{code: true})
			case "separate":
				if len(stack) > 0 {
					current := stack[len(stack)-1]
					if referenceField(current.instruction.String()) {
						return fmt.Errorf("closing-page reference fields are unsupported")
					}
					current.code = false
				}
			case "end":
				if len(stack) == 0 {
					return fmt.Errorf("unbalanced closing field")
				}
				current := stack[len(stack)-1]
				if referenceField(current.instruction.String()) {
					return fmt.Errorf("closing-page reference fields are unsupported")
				}
				stack = stack[:len(stack)-1]
			}
		}
		if element.Tag == "instrText" && len(stack) > 0 && stack[len(stack)-1].code {
			stack[len(stack)-1].instruction.WriteString(element.Text())
		}
	}
	if len(stack) != 0 {
		return fmt.Errorf("unbalanced closing field")
	}
	return nil
}
