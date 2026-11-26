package editing

import (
	"bytes"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

func decodeLogo(data []byte, requirePNG bool) (image.Image, error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err == image.ErrFormat && !requirePNG {
		return decodeSVGLogo(data)
	}
	if err != nil || (requirePNG && format != "png") || config.Width < 1 ||
		config.Height < 1 || config.Width > maxLogoInputSide || config.Height > maxLogoInputSide {
		return nil, errLogo
	}
	switch format {
	case "png", "jpeg", "gif", "bmp", "tiff", "webp":
	default:
		// The contract is independent of decoders registered by callers.
		return nil, errLogo
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errLogo
	}
	if format == "gif" {
		// The first frame may occupy only part of the declared logical screen.
		// Keep its offset and the transparent space outside it.
		screen := image.Rect(0, 0, config.Width, config.Height)
		if decoded.Bounds() != screen {
			frame := image.NewNRGBA(screen)
			draw.Draw(frame, decoded.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
			decoded = frame
		}
	}
	return decoded, nil
}
