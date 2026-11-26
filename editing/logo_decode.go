package editing

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

func decodeLogo(data []byte, requirePNG bool) (image.Image, error) {
	var config func(io.Reader) (image.Config, error)
	var decode func(io.Reader) (image.Image, error)
	isGIF := false
	// An application's registered image decoders must never intercept logo
	// data before this package validates its type and dimensions.
	switch {
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		config, decode = png.DecodeConfig, png.Decode
	case requirePNG:
		return nil, errLogo
	case bytes.HasPrefix(data, []byte("\xff\xd8")):
		config, decode = jpeg.DecodeConfig, jpeg.Decode
	case bytes.HasPrefix(data, []byte("GIF87a")), bytes.HasPrefix(data, []byte("GIF89a")):
		config, decode, isGIF = gif.DecodeConfig, gif.Decode, true
	case bytes.HasPrefix(data, []byte("BM")):
		config, decode = bmp.DecodeConfig, bmp.Decode
	case bytes.HasPrefix(data, []byte("II*\x00")), bytes.HasPrefix(data, []byte("MM\x00*")):
		config, decode = tiff.DecodeConfig, tiff.Decode
	case len(data) >= 12 && bytes.HasPrefix(data, []byte("RIFF")) && string(data[8:12]) == "WEBP":
		config, decode = webp.DecodeConfig, webp.Decode
	default:
		return decodeSVGLogo(data)
	}
	size, err := config(bytes.NewReader(data))
	if err != nil || size.Width < 1 || size.Height < 1 ||
		size.Width > maxLogoInputSide || size.Height > maxLogoInputSide {
		return nil, errLogo
	}
	decoded, err := decode(bytes.NewReader(data))
	if err != nil {
		return nil, errLogo
	}
	if isGIF {
		// The first frame may occupy only part of the declared logical screen.
		// Keep its offset and the transparent space outside it.
		screen := image.Rect(0, 0, size.Width, size.Height)
		if decoded.Bounds() != screen {
			frame := image.NewNRGBA(screen)
			draw.Draw(frame, decoded.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
			decoded = frame
		}
	}
	return decoded, nil
}
