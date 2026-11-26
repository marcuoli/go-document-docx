package editing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"

	"github.com/marcuoli/go-document-docx/document"
	"github.com/mmonterroca/docxgo/v2/domain"
	resample "golang.org/x/image/draw"
)

const (
	maxLogoInputBytes = 8 << 20
	maxLogoInputSide  = 4096
	logoRasterSide    = 640
)

var errLogo = errors.New("logo requires a readable PNG, JPEG, GIF, BMP, TIFF, WebP or self-contained SVG up to 8 MiB, with raster dimensions up to 4096 pixels per side")

// Logo is an immutable, normalized image for document artwork. A zero Logo
// leaves the supplied document or upper artwork unmodified.
type Logo struct {
	png           []byte
	width, height int
}

// PNG returns a caller-owned copy of the prepared PNG and its original width
// and height in pixels before resampling. The PNG may have smaller dimensions.
// A zero Logo returns nil and zero dimensions.
func (logo Logo) PNG() (data []byte, originalWidth, originalHeight int) {
	return bytes.Clone(logo.png), logo.width, logo.height
}

// LogoOptions controls optional transformations during logo preparation.
// Its zero value preserves the supplied background.
type LogoOptions struct {
	// RemoveWhiteBackground clears near-white pixels connected to an image edge.
	// Existing alpha, enclosed white areas and entirely white images are preserved.
	RemoveWhiteBackground bool
}

// ReadLogo prepares a bounded PNG, JPEG, GIF, BMP, TIFF, WebP or static SVG
// from an existing reader. It preserves the supplied background and alpha,
// uses the first GIF frame and normalizes to PNG without source metadata.
// SVG supports self-contained shapes, outline paths and gradients. ReadLogo
// never closes the reader.
func ReadLogo(r io.Reader) (Logo, error) {
	return ReadLogoWithOptions(r, LogoOptions{})
}

// ReadLogoWithOptions prepares a logo using the supplied options before
// resampling. It follows ReadLogo's limits and never closes the reader.
func ReadLogoWithOptions(r io.Reader, options LogoOptions) (Logo, error) {
	if r == nil {
		return Logo{}, errLogo
	}
	data, err := io.ReadAll(io.LimitReader(r, maxLogoInputBytes+1))
	if err != nil || len(data) > maxLogoInputBytes {
		return Logo{}, errLogo
	}
	asset, err := decodeLogo(data, false)
	if err != nil {
		return Logo{}, err
	}
	if options.RemoveWhiteBackground {
		asset = removeWhiteLogoBackground(asset)
	}
	width, height := asset.Bounds().Dx(), asset.Bounds().Dy()
	if max(width, height) > logoRasterSide {
		w, h := fitLogo(width, height, logoRasterSide)
		// A one-pixel side cannot be sampled any smaller. Keep the original
		// raster here so its representable aspect ratio is preserved.
		if min(w, h) > 1 {
			raster := image.NewNRGBA(image.Rect(0, 0, w, h))
			resample.CatmullRom.Scale(raster, raster.Bounds(), asset, asset.Bounds(), draw.Src, nil)
			asset = raster
		}
	}
	data, err = encodeLogo(asset)
	if err != nil {
		return Logo{}, err
	}
	return Logo{png: data, width: width, height: height}, nil
}

func fitLogo(width, height, side int) (int, int) {
	if width >= height {
		return side, max(1, height*side/width)
	}
	return max(1, width*side/height), side
}

func encodeLogo(pic image.Image) ([]byte, error) {
	var output bytes.Buffer
	if err := png.Encode(&output, pic); err != nil {
		return nil, fmt.Errorf("encode logo artwork: %w", err)
	}
	return output.Bytes(), nil
}

// LogoCell selects an existing table cell with zero-based indices. SideTwips
// is the outer square for the image. SpaceBeforeTwips precedes the new paragraph.
type LogoCell struct {
	Table, Row, Column          int
	SideTwips, SpaceBeforeTwips int
}

// CenterInTableCell appends one centered, aspect-preserving logo paragraph to
// a selected cell. It reads and serializes a private document copy, leaves the
// original unchanged on failure, and returns its input untouched for a zero logo.
func (logo Logo) CenterInTableCell(ctx context.Context, input []byte, options document.Options, placement LogoCell) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(logo.png) == 0 {
		return input, nil
	}
	if placement.Table < 0 || placement.Row < 0 || placement.Column < 0 || placement.SideTwips < 1 ||
		placement.SideTwips > 31680 || placement.SpaceBeforeTwips < 0 || placement.SpaceBeforeTwips > 31680 {
		return nil, errors.New("invalid logo placement")
	}
	doc, err := Open(ctx, input, options)
	if err != nil {
		return nil, err
	}
	tables := doc.Native().Tables()
	if placement.Table >= len(tables) {
		return nil, errors.New("logo table is missing")
	}
	row, err := tables[placement.Table].Row(placement.Row)
	if err != nil {
		return nil, err
	}
	cell, err := row.Cell(placement.Column)
	if err != nil {
		return nil, err
	}
	paragraph, err := cell.AddParagraph()
	if err != nil {
		return nil, err
	}
	w, h := fitLogo(logo.width, logo.height, placement.SideTwips)
	if err := errors.Join(paragraph.SetAlignment(domain.AlignmentCenter),
		paragraph.SetSpacingBefore(placement.SpaceBeforeTwips), AddImage(paragraph, logo.png, w, h)); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := doc.Write(ctx, &output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// StackBelowPNG returns existing PNG artwork with the logo centered beneath
// it. Both side and gap are measured in pixels. A zero logo returns upper
// untouched. The result stays within the original width and 4096 pixels high.
func (logo Logo) StackBelowPNG(upper []byte, side, gap int) ([]byte, error) {
	if len(logo.png) == 0 {
		return upper, nil
	}
	if side < 1 || side > maxLogoInputSide || gap < 0 || gap > maxLogoInputSide || len(upper) > maxLogoInputBytes {
		return nil, errors.New("invalid logo artwork bounds")
	}
	brand, err := decodeLogo(upper, true)
	if err != nil {
		return nil, err
	}
	b := brand.Bounds()
	w, h := fitLogo(logo.width, logo.height, min(side, b.Dx()))
	top := b.Dy() + gap
	if top+h > maxLogoInputSide {
		return nil, errors.New("combined logo artwork exceeds 4096 pixels in height")
	}
	pic, err := png.Decode(bytes.NewReader(logo.png))
	if err != nil {
		return nil, err
	}
	result := image.NewNRGBA(image.Rect(0, 0, b.Dx(), top+h))
	draw.Draw(result, image.Rect(0, 0, b.Dx(), b.Dy()), brand, b.Min, draw.Src)
	left := (b.Dx() - w) / 2
	resample.CatmullRom.Scale(result, image.Rect(left, top, left+w, top+h), pic, pic.Bounds(), draw.Src, nil)
	return encodeLogo(result)
}
