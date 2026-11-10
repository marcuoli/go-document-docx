package editing

import (
	"errors"
	"fmt"

	docx "github.com/mmonterroca/docxgo/v2"
	"github.com/mmonterroca/docxgo/v2/domain"
)

// ParagraphStyle creates a named derived style with explicit family and size in
// half-points. The caller may use the returned native style for further supported
// formatting. IDs must be unique; a missing parent or self-reference is an error.
func (d *Document) ParagraphStyle(id, name, basedOn, font string, size int) (domain.ParagraphStyle, error) {
	for _, s := range []string{id, name, basedOn, font} {
		if err := validText(s); err != nil {
			return nil, err
		}
	}
	if id == "" || name == "" || len(id) > 253 || id == basedOn {
		return nil, fmt.Errorf("invalid paragraph style identity")
	}
	if d.native.StyleManager().HasStyle(id) {
		return nil, fmt.Errorf("paragraph style already exists")
	}
	if basedOn != "" && !d.native.StyleManager().HasStyle(basedOn) {
		return nil, fmt.Errorf("parent style not found")
	}
	if font == "" || size == 0 {
		return nil, fmt.Errorf("derived style family and size are required")
	}
	if err := validateStyle(TextStyle{Font: font, Size: size}); err != nil {
		return nil, err
	}
	style := docx.NewParagraphStyle(id, name)
	err := style.SetBasedOn(basedOn)
	if font != "" {
		err = errors.Join(err, style.SetFont(domain.Font{Name: font}))
	}
	if size != 0 {
		err = errors.Join(err, style.SetSize(size))
	}
	if err != nil {
		return nil, err
	}
	if err := d.native.StyleManager().AddStyle(style); err != nil {
		return nil, err
	}
	d.styles = append(d.styles, style)
	return style, nil
}
