package document

import (
	"encoding/hex"
	"fmt"
)

// TableStyle configures literal table appearance. Colors are six-digit RGB;
// empty colors/fonts retain the literal renderer's defaults. Padding and
// ParagraphAfter are explicit twips, including zero. VerticalAlignment accepts
// top, center or bottom (empty uses top). GridBorders adds all six grid borders.
// New takes a copy; later changes to this value cannot alter the document.
type TableStyle struct {
	HeaderFill, HeaderTextColor, HeaderFont string
	BodyFill, AlternateFill, BorderColor    string
	PaddingTop, PaddingLeft                 int
	PaddingBottom, PaddingRight             int
	ParagraphAfter                          int
	VerticalAlignment                       string
	GridBorders                             bool
}

func tableStyleFor(o Options) TableStyle {
	s := TableStyle{
		HeaderFill: "E9E5DF", HeaderFont: o.SubheadingFont,
		BodyFill: "FFFFFF", AlternateFill: "F7F6F4", BorderColor: "D6D2CE",
		PaddingTop: 70, PaddingLeft: 108, PaddingBottom: 70, PaddingRight: 108,
		ParagraphAfter: 30, VerticalAlignment: "top",
	}
	if o.TableStyle == nil {
		return s
	}
	requested := *o.TableStyle
	if requested.HeaderFill == "" {
		requested.HeaderFill = s.HeaderFill
	}
	if requested.HeaderFont == "" {
		requested.HeaderFont = s.HeaderFont
	}
	if requested.BodyFill == "" {
		requested.BodyFill = s.BodyFill
	}
	if requested.AlternateFill == "" {
		requested.AlternateFill = s.AlternateFill
	}
	if requested.BorderColor == "" {
		requested.BorderColor = s.BorderColor
	}
	if requested.VerticalAlignment == "" {
		requested.VerticalAlignment = "top"
	}
	return requested
}

func validateTableStyle(s TableStyle) error {
	for _, color := range []string{s.HeaderFill, s.HeaderTextColor, s.BodyFill, s.AlternateFill, s.BorderColor} {
		if color == "" {
			continue
		}
		if _, err := hex.DecodeString(color); len(color) != 6 || err != nil {
			return fmt.Errorf("table colors must be six-digit RGB")
		}
	}
	for _, twips := range []int{s.PaddingTop, s.PaddingLeft, s.PaddingBottom, s.PaddingRight, s.ParagraphAfter} {
		if twips < 0 || twips > 31680 {
			return fmt.Errorf("table padding/spacing exceeds supported bounds")
		}
	}
	if len(s.HeaderFont) > 255 {
		return fmt.Errorf("table header font exceeds 255 bytes")
	}
	if s.VerticalAlignment != "" && s.VerticalAlignment != "top" && s.VerticalAlignment != "center" && s.VerticalAlignment != "bottom" {
		return fmt.Errorf("invalid table vertical alignment")
	}
	return nil
}
