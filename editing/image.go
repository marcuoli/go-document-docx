package editing

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/mmonterroca/docxgo/v2/domain"
)

// Image appends a caller-provided PNG, JPEG or GIF to this paragraph. Width and
// height use twips; one zero dimension preserves aspect ratio. Both zero use 96 dpi.
// No filesystem/URL lookup or asset discovery occurs. Bytes are bounded and copied
// by the upstream image manager. Pixel limits prevent oversized declared images.
func (p *Paragraph) Image(data []byte, width, height int) error {
	return AddImage(p.native, data, width, height)
}

// AddImage appends caller-supplied image bytes to an upstream paragraph, including
// a paragraph loaded from a template. It uses the same validation as Paragraph.Image.
func AddImage(paragraph domain.Paragraph, data []byte, width, height int) error {
	if len(data) == 0 || len(data) > 32<<20 || width < 0 || height < 0 || width > 31680 || height > 31680 {
		return fmt.Errorf("invalid image size")
	}
	cfg, kind, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return err
	}
	if cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 100_000_000 {
		return fmt.Errorf("image dimensions exceed limit")
	}
	if width == 0 && height == 0 {
		width = cfg.Width * 15
		height = cfg.Height * 15
	} else if width == 0 {
		width = int(int64(height) * int64(cfg.Width) / int64(cfg.Height))
	} else if height == 0 {
		height = int(int64(width) * int64(cfg.Height) / int64(cfg.Width))
	}
	if width < 1 || height < 1 || width > 31680 || height > 31680 {
		return fmt.Errorf("image does not fit supported dimensions")
	}
	format := domain.ImageFormatPNG
	switch kind {
	case "jpeg":
		format = domain.ImageFormatJPEG
	case "gif":
		format = domain.ImageFormatGIF
	}
	_, err = paragraph.AddImageFromBytesWithSize(bytes.Clone(data), format, domain.NewImageSizeInches(float64(width)/1440, float64(height)/1440))
	return err
}
