package opc

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/beevik/etree"
)

func xmlAdmission(b []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(b))
	depth, nodes, roots := 0, 0, 0
	for {
		token, err := decoder.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch t := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				roots++
				if roots > 1 {
					return fmt.Errorf("multiple XML roots")
				}
			}
			depth++
			nodes++
			if depth > 128 || nodes > 2_000_000 || len(t.Attr) > 256 {
				return fmt.Errorf("XML structure exceeds supported limits")
			}
		case xml.EndElement:
			depth--
			if depth < 0 {
				return fmt.Errorf("unbalanced XML")
			}
		case xml.Directive:
			return fmt.Errorf("XML directives are unsupported")
		}
	}
	if depth != 0 {
		return fmt.Errorf("unbalanced XML")
	}
	return nil
}
func validateTree(root *etree.Element) error {
	pending := []*etree.Element{root}
	nodes := 0
	for len(pending) > 0 {
		e := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		nodes++
		if nodes > 2_000_000 {
			return fmt.Errorf("XML structure exceeds supported limits")
		}
		if e.Space != "" && e.NamespaceURI() == "" {
			return fmt.Errorf("unbound XML element prefix")
		}
		for _, a := range e.Attr {
			if a.Space != "" && a.Space != "xmlns" && a.Space != "xml" && a.NamespaceURI() == "" {
				return fmt.Errorf("unbound XML attribute prefix")
			}
			if err := Text(a.Value); err != nil {
				return err
			}
		}
		for _, token := range e.Child {
			if text, ok := token.(*etree.CharData); ok {
				if err := Text(text.Data); err != nil {
					return err
				}
			}
		}
		pending = append(pending, e.ChildElements()...)
	}
	return nil
}
