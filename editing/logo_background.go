package editing

import (
	"image"
	imagecolor "image/color"
)

func removeWhiteLogoBackground(pic image.Image) image.Image {
	// Supported PNG and JPEG decoder types can report whether any pixel already
	// has alpha. Keep that artist-supplied mask, including white artwork at an edge.
	opacity, ok := pic.(interface{ Opaque() bool })
	if !ok || !opacity.Opaque() {
		return pic
	}

	bounds := pic.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	mask := make([]bool, width*height)
	var pending []int32 // The decoded image is bounded to 4,096 pixels per side.
	cleared := 0
	visit := func(pixel int) {
		if mask[pixel] {
			return
		}
		r, g, b, _ := pic.At(bounds.Min.X+pixel%width, bounds.Min.Y+pixel/width).RGBA()
		// Near-neutral JPEG compression noise should not leave isolated matte specks.
		const minWhite = 248 * 257
		const maxSpread = 4 * 257
		low, high := min(r, g, b), max(r, g, b)
		if low >= minWhite && high-low <= maxSpread {
			mask[pixel] = true
			cleared++
			pending = append(pending, int32(pixel))
		}
	}
	for x := range width {
		visit(x)
		visit((height-1)*width + x)
	}
	for y := 1; y < height-1; y++ {
		visit(y * width)
		visit((y+1)*width - 1)
	}
	for len(pending) != 0 {
		pixel := int(pending[len(pending)-1])
		pending = pending[:len(pending)-1]
		x := pixel % width
		if x > 0 {
			visit(pixel - 1)
		}
		if x+1 < width {
			visit(pixel + 1)
		}
		if pixel >= width {
			visit(pixel - width)
		}
		if pixel+width < len(mask) {
			visit(pixel + width)
		}
	}
	// Without foreground, a matte and a pale image are indistinguishable.
	if cleared == 0 || cleared == len(mask) {
		return pic
	}
	return logoAlphaMask{Image: pic, cleared: mask}
}

// The mask changes only selected alpha values and exposes the original color
// precision to the PNG encoder. It shares the private decoded image read-only.
type logoAlphaMask struct {
	image.Image
	cleared []bool
}

func (m logoAlphaMask) ColorModel() imagecolor.Model {
	model := m.Image.ColorModel()
	if _, indexed := model.(imagecolor.Palette); indexed {
		return imagecolor.NRGBAModel
	}
	switch model {
	case imagecolor.GrayModel, imagecolor.RGBAModel, imagecolor.NRGBAModel, imagecolor.AlphaModel:
		return imagecolor.NRGBAModel
	default:
		// PNG also uses 16 bits for other models, including decoded JPEG.
		return imagecolor.NRGBA64Model
	}
}

func (m logoAlphaMask) At(x, y int) imagecolor.Color {
	b := m.Bounds()
	if image.Pt(x, y).In(b) && m.cleared[(y-b.Min.Y)*b.Dx()+x-b.Min.X] {
		return imagecolor.Transparent
	}
	return m.Image.At(x, y)
}
