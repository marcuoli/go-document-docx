// Package opc owns bounded ZIP/XML operations on document artifacts, not dependency source.
// It never extracts parts to disk, resolves external links, or loads XML entities.
package opc

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/beevik/etree"
)

// Word, Relations, Office and Types are the namespaces used by admitted DOCX parts.
// MaxPackage and MaxPart bound uncompressed package and individual part bytes.
const (
	Word       = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	Relations  = "http://schemas.openxmlformats.org/package/2006/relationships"
	Office     = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	Types      = "http://schemas.openxmlformats.org/package/2006/content-types"
	MaxPackage = 256 << 20
	MaxPart    = 128 << 20
)

// Package owns uncompressed parts. Callers must use it serially.
type Package struct{ Parts map[string][]byte }

// Read admits a bounded DOCX ZIP into owned parts without extracting files.
func Read(ctx context.Context, data []byte) (*Package, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(data) > 64<<20 {
		return nil, fmt.Errorf("compressed document exceeds 64 MiB")
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("document ZIP: %w", err)
	}
	if len(z.File) > 4096 {
		return nil, fmt.Errorf("too many package parts")
	}
	p := &Package{Parts: map[string][]byte{}}
	total := 0
	for _, f := range z.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if f.Name == "" || path.Clean(f.Name) != f.Name || strings.ContainsAny(f.Name, "\\:") || strings.HasPrefix(f.Name, "/") || strings.HasPrefix(f.Name, "../") {
			return nil, fmt.Errorf("invalid package part name")
		}
		if _, ok := p.Parts[f.Name]; ok {
			return nil, fmt.Errorf("duplicate package part")
		}
		if f.UncompressedSize64 > MaxPart || f.UncompressedSize64 > uint64(MaxPackage-total) {
			return nil, fmt.Errorf("document size limit exceeded")
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, readErr := io.ReadAll(io.LimitReader(contextReader{ctx, r}, MaxPart+1))
		err = errors.Join(readErr, r.Close())
		if err != nil {
			return nil, err
		}
		total += len(b)
		if len(b) > MaxPart || total > MaxPackage {
			return nil, fmt.Errorf("document size limit exceeded")
		}
		p.Parts[f.Name] = b
	}
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml"} {
		if _, ok := p.Parts[name]; !ok {
			return nil, fmt.Errorf("missing document part %s", name)
		}
	}
	return p, nil
}

// XML parses an existing part with namespace, depth, size and character checks.
func (p *Package) XML(name string) (*etree.Document, error) {
	b, ok := p.Parts[name]
	if !ok {
		return nil, fmt.Errorf("missing XML part %s", name)
	}
	if len(b) > MaxPart || bytes.Contains(bytes.ToUpper(b), []byte("<!DOCTYPE")) {
		return nil, fmt.Errorf("unsupported or oversized XML part")
	}
	if err := xmlAdmission(b); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	d := etree.NewDocument()
	if err := d.ReadFromBytes(b); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if d.Root() == nil {
		return nil, fmt.Errorf("empty XML part %s", name)
	}
	if err := validateTree(d.Root()); err != nil {
		return nil, err
	}
	if d.Root().NamespaceURI() == Word {
		for prefix, expected := range map[string]string{"w": Word, "r": Office} {
			if declared := d.Root().SelectAttrValue("xmlns:"+prefix, ""); declared != "" && declared != expected {
				return nil, fmt.Errorf("unsupported reserved XML prefix binding")
			}
		}
	}
	return d, nil
}

// SetXML validates and replaces an owned XML part; failure preserves its bytes.
func (p *Package) SetXML(name string, d *etree.Document) error {
	if d == nil || d.Root() == nil {
		return fmt.Errorf("XML root is required")
	}
	if err := validateTree(d.Root()); err != nil {
		return err
	}
	b, err := d.WriteToBytes()
	if err != nil {
		return err
	}
	if len(b) > MaxPart {
		return fmt.Errorf("XML part exceeds limit")
	}
	if err := xmlAdmission(b); err != nil {
		return err
	}
	p.Parts[name] = b
	return nil
}

// Write serializes owned parts with bounded output and cancellation; errors can leave partial output.
func (p *Package) Write(ctx context.Context, w io.Writer) error {
	if w == nil {
		return fmt.Errorf("writer is required")
	}
	total := 0
	if len(p.Parts) > 4096 {
		return fmt.Errorf("too many package parts")
	}
	names := make([]string, 0, len(p.Parts))
	for name, b := range p.Parts {
		if len(b) > MaxPart || len(b) > MaxPackage-total {
			return fmt.Errorf("document size limit exceeded")
		}
		total += len(b)
		names = append(names, name)
	}
	slices.Sort(names)
	z := zip.NewWriter(Limited(ctx, w, 64<<20))
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return errors.Join(err, z.Close())
		}
		f, err := z.Create(name)
		if err == nil {
			_, err = f.Write(p.Parts[name])
		}
		if err != nil {
			return errors.Join(err, z.Close())
		}
	}
	return errors.Join(z.Close(), ctx.Err())
}

// Writer checks cancellation and io.Writer's short-write contract.
type Writer struct {
	Ctx         context.Context
	Destination io.Writer
}

// Write serializes owned parts with bounded output and cancellation; errors can leave partial output.
func (w Writer) Write(p []byte) (int, error) {
	if err := w.Ctx.Err(); err != nil {
		return 0, err
	}
	n, err := w.Destination.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

// Read admits a bounded DOCX ZIP into owned parts without extracting files.
func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

// Elements returns matching descendants, including root, in document order.
func Elements(root *etree.Element, namespace, tag string) []*etree.Element {
	var out []*etree.Element
	var walk func(*etree.Element)
	walk = func(e *etree.Element) {
		if e.Tag == tag && e.NamespaceURI() == namespace {
			out = append(out, e)
		}
		for _, c := range e.ChildElements() {
			walk(c)
		}
	}
	walk(root)
	return out
}

// Child returns the first direct child with the specified namespace and local name.
func Child(root *etree.Element, namespace, tag string) *etree.Element {
	for _, e := range root.ChildElements() {
		if e.Tag == tag && e.NamespaceURI() == namespace {
			return e
		}
	}
	return nil
}

// Attr returns a namespace-qualified attribute value or an empty string when absent.
func Attr(e *etree.Element, namespace, tag string) string {
	for _, a := range e.Attr {
		if a.Key == tag && a.NamespaceURI() == namespace {
			return a.Value
		}
	}
	return ""
}

// SetAttr updates an attribute by namespace and local name, retaining its prefix.
// New attributes use the canonical Word prefix; callers must establish its binding.
func SetAttr(e *etree.Element, namespace, key, value string) {
	for i, a := range e.Attr {
		if a.Key == key && a.NamespaceURI() == namespace {
			e.Attr[i].Value = value
			return
		}
	}
	e.CreateAttr("w:"+key, value)
}

// Ensure returns a direct Word child or appends one using the established w binding.
func Ensure(root *etree.Element, tag string) *etree.Element {
	if e := Child(root, Word, tag); e != nil {
		return e
	}
	return root.CreateElement("w:" + tag)
}

// WordPrefix establishes the reserved Word and office relationship bindings on root.
func WordPrefix(root *etree.Element) {
	root.CreateAttr("xmlns:w", Word)
	root.CreateAttr("xmlns:r", Office)
}

// RelName returns the relationship part name for an admitted package part.
func RelName(part string) string { return path.Join(path.Dir(part), "_rels", path.Base(part)+".rels") }

// Resolve resolves an internal relationship target and rejects package escapes.
func Resolve(part, target string) (string, error) {
	if strings.ContainsAny(target, "\\?#") || target == "" {
		return "", fmt.Errorf("unsupported relationship target")
	}
	n := path.Clean(path.Join(path.Dir(part), target))
	if strings.HasPrefix(target, "/") {
		n = path.Clean(strings.TrimPrefix(target, "/"))
	}
	if n == "." || strings.HasPrefix(n, "../") || strings.Contains(n, ":") {
		return "", fmt.Errorf("relationship escapes package")
	}
	return n, nil
}

// RelationshipDocument creates an empty package relationship document.
func RelationshipDocument() *etree.Document {
	d := etree.NewDocument()
	d.CreateElement("Relationships").CreateAttr("xmlns", Relations)
	return d
}

// Relationships reads a part relationship document, or creates an empty one when absent.
func (p *Package) Relationships(part string) (*etree.Document, error) {
	name := RelName(part)
	if _, ok := p.Parts[name]; !ok {
		return RelationshipDocument(), nil
	}
	d, err := p.XML(name)
	if err != nil {
		return nil, err
	}
	if d.Root().Tag != "Relationships" || d.Root().NamespaceURI() != Relations {
		return nil, fmt.Errorf("invalid relationship namespace")
	}
	return d, nil
}

// AddRelation appends an office relationship with a unique ID and returns that ID.
func AddRelation(d *etree.Document, kind, target, mode string) string {
	used := map[string]bool{}
	for _, e := range d.Root().ChildElements() {
		used[e.SelectAttrValue("Id", "")] = true
	}
	id := ""
	for i := 1; ; i++ {
		id = fmt.Sprintf("rId%d", i)
		if !used[id] {
			break
		}
	}
	r := d.Root().CreateElement("Relationship")
	r.CreateAttr("xmlns", Relations)
	r.CreateAttr("Id", id)
	r.CreateAttr("Type", Office+"/"+kind)
	r.CreateAttr("Target", target)
	if mode != "" {
		r.CreateAttr("TargetMode", mode)
	}
	return id
}

// ContentType adds or replaces a content type override for an owned package part.
func (p *Package) ContentType(part, kind string) error {
	d, err := p.XML("[Content_Types].xml")
	if err != nil {
		return err
	}
	for _, e := range d.Root().ChildElements() {
		if e.Tag == "Override" && e.SelectAttrValue("PartName", "") == "/"+part {
			e.CreateAttr("ContentType", kind)
			return p.SetXML("[Content_Types].xml", d)
		}
	}
	e := d.Root().CreateElement("Override")
	e.CreateAttr("xmlns", Types)
	e.CreateAttr("PartName", "/"+part)
	e.CreateAttr("ContentType", kind)
	return p.SetXML("[Content_Types].xml", d)
}

// NewXML creates a Word document with the given root local name and reserved bindings.
func NewXML(root string) *etree.Document {
	d := etree.NewDocument()
	e := d.CreateElement("w:" + root)
	WordPrefix(e)
	return d
}

// Text rejects invalid UTF-8 and XML 1.0 code points before any escaping occurs.
func Text(text string) error {
	if !utf8.ValidString(text) {
		return fmt.Errorf("text is not UTF-8")
	}
	for _, r := range text {
		if !(r == 9 || r == 10 || r == 13 || r >= 32 && r <= 0xd7ff || r >= 0xe000 && r <= 0xfffd || r >= 0x10000 && r <= 0x10ffff) {
			return fmt.Errorf("text contains XML-invalid characters")
		}
	}
	return nil
}

// Relative returns a slash-separated package target from one part to another.
// Inputs are admitted canonical part names; filesystem paths are never involved.
func Relative(part, target string) string {
	from := strings.Split(path.Dir(part), "/")
	if path.Dir(part) == "." {
		from = nil
	}
	to := strings.Split(target, "/")
	i := 0
	for i < len(from) && i < len(to) && from[i] == to[i] {
		i++
	}
	components := []string{}
	for range from[i:] {
		components = append(components, "..")
	}
	components = append(components, to[i:]...)
	return strings.Join(components, "/")
}
