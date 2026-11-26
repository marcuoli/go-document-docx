package editing

import (
	"bytes"
	"image"
	imagecolor "image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestReadLogoWithOptionsRemovesWhiteBackground(t *testing.T) {
	const size = 80
	const matte = uint8(252)
	white := imagecolor.NRGBA{R: matte, G: matte, B: matte, A: 255}
	blue := imagecolor.NRGBA{R: 30, G: 90, B: 160, A: 255}
	source := image.NewNRGBA(image.Rect(0, 0, size, size))
	draw.Draw(source, source.Bounds(), image.NewUniform(white), image.Point{}, draw.Src)
	draw.Draw(source, image.Rect(16, 16, 64, 64), image.NewUniform(blue), image.Point{}, draw.Src)
	draw.Draw(source, image.Rect(30, 30, 50, 50), image.NewUniform(white), image.Point{}, draw.Src)

	for _, format := range []string{"PNG", "JPEG"} {
		t.Run(format, func(t *testing.T) {
			var raw bytes.Buffer
			var err error
			if format == "JPEG" {
				err = jpeg.Encode(&raw, source, &jpeg.Options{Quality: 95})
			} else {
				err = png.Encode(&raw, source)
			}
			if err != nil {
				t.Fatal(err)
			}
			legacy, err := ReadLogo(bytes.NewReader(raw.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			unchanged, err := ReadLogoWithOptions(bytes.NewReader(raw.Bytes()), LogoOptions{})
			if err != nil || !bytes.Equal(unchanged.png, legacy.png) {
				t.Fatalf("zero options changed the legacy result: %v", err)
			}
			after := prepareLogoWithRemoval(t, raw.Bytes())
			before, err := png.Decode(bytes.NewReader(legacy.png))
			if err != nil {
				t.Fatal(err)
			}
			if after.Bounds() != source.Bounds() {
				t.Fatal("background removal changed the logo dimensions")
			}
			if _, _, _, alpha := after.At(3, 3).RGBA(); alpha != 0 {
				t.Fatal("near-white background remained opaque")
			}
			for _, point := range []image.Point{image.Pt(40, 40), image.Pt(24, 24)} {
				r0, g0, b0, a0 := before.At(point.X, point.Y).RGBA()
				r1, g1, b1, a1 := after.At(point.X, point.Y).RGBA()
				if r1 != r0 || g1 != g0 || b1 != b0 || a1 != a0 || a1 != 0xffff {
					t.Errorf("artwork changed at %v", point)
				}
			}
		})
	}
}

func TestRemoveWhiteLogoBackgroundProtectsExistingAlphaAndPrecision(t *testing.T) {
	white := imagecolor.NRGBA64{R: 0xffff, G: 0xffff, B: 0xffff, A: 0xffff}
	source := image.NewNRGBA64(image.Rect(0, 0, 5, 5))
	draw.Draw(source, source.Bounds(), image.NewUniform(white), image.Point{}, draw.Src)
	encode := func() []byte {
		var raw bytes.Buffer
		if err := png.Encode(&raw, source); err != nil {
			t.Fatal(err)
		}
		return raw.Bytes()
	}
	decodePixel := func(raw []byte, x, y int) imagecolor.NRGBA64 {
		return imagecolor.NRGBA64Model.Convert(prepareLogoWithRemoval(t, raw).At(x, y)).(imagecolor.NRGBA64)
	}

	if got := decodePixel(encode(), 0, 0); got != white {
		t.Fatal("entirely white image was erased")
	}
	color16 := imagecolor.NRGBA64{R: 0x1234, G: 0x5678, B: 0x9abc, A: 0xffff}
	source.SetNRGBA64(2, 2, color16)
	raw := encode()
	if got := decodePixel(raw, 0, 0); got.A != 0 {
		t.Fatal("opaque 16-bit matte was not cleared")
	}
	if got := decodePixel(raw, 2, 2); got != color16 {
		t.Fatalf("preserved artwork lost color precision: %#v", got)
	}

	color16.A = 0x6070
	source.SetNRGBA64(2, 2, color16)
	raw = encode()
	if got := decodePixel(raw, 0, 0); got != white {
		t.Fatal("logo that already had alpha lost an opaque white edge")
	}
	if got := decodePixel(raw, 2, 2); got != color16 {
		t.Fatalf("artwork lost existing partial alpha or color precision: %#v", got)
	}
}

func prepareLogoWithRemoval(t *testing.T, input []byte) image.Image {
	t.Helper()
	logo, err := ReadLogoWithOptions(bytes.NewReader(input), LogoOptions{RemoveWhiteBackground: true})
	if err != nil {
		t.Fatal(err)
	}
	pic, err := png.Decode(bytes.NewReader(logo.png))
	if err != nil {
		t.Fatal(err)
	}
	return pic
}
