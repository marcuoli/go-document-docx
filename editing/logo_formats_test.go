package editing

import (
	"bytes"
	"encoding/base64"
	"image"
	imagecolor "image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

func TestLogoRasterFormats(t *testing.T) {
	palette := imagecolor.Palette{imagecolor.Transparent, imagecolor.RGBA{R: 190, G: 20, B: 80, A: 255}}
	pic := image.NewPaletted(image.Rect(0, 0, 8, 4), palette)
	pic.SetColorIndex(3, 2, 1)
	encoders := []struct {
		name   string
		encode func(io.Writer) error
		alpha  bool
	}{
		{"PNG", func(w io.Writer) error { return png.Encode(w, pic) }, true},
		{"JPEG", func(w io.Writer) error { return jpeg.Encode(w, pic, nil) }, false},
		{"BMP", func(w io.Writer) error { return bmp.Encode(w, pic) }, false},
		{"TIFF", func(w io.Writer) error {
			alpha := image.NewNRGBA(pic.Bounds())
			alpha.Set(3, 2, palette[1])
			return tiff.Encode(w, alpha, nil)
		}, true},
		{"WebP", func(w io.Writer) error {
			data, err := base64.StdEncoding.DecodeString("UklGRiAAAABXRUJQVlA4TBQAAAAvB8AAEA8wFOM7UPMf8BiEiP4HAQ==")
			if err != nil {
				return err
			}
			_, err = w.Write(data)
			return err
		}, true},
	}
	for _, tc := range encoders {
		t.Run(tc.name, func(t *testing.T) {
			var input bytes.Buffer
			if err := tc.encode(&input); err != nil {
				t.Fatal(err)
			}
			logo, err := ReadLogo(&input)
			if err != nil {
				t.Fatal(err)
			}
			if logo.width != 8 || logo.height != 4 {
				t.Fatalf("normalized size = %d x %d", logo.width, logo.height)
			}
			decoded, err := png.Decode(bytes.NewReader(logo.png))
			if err != nil {
				t.Fatal(err)
			}
			if tc.alpha {
				_, _, _, corner := decoded.At(0, 0).RGBA()
				_, _, _, paint := decoded.At(3, 2).RGBA()
				if corner != 0 || paint != 0xffff {
					t.Fatalf("alpha = %x at corner, %x at paint", corner, paint)
				}
			}
		})
	}
}

func TestLogoGIFUsesFirstFrameAndLogicalScreen(t *testing.T) {
	frame := image.NewPaletted(image.Rect(2, 1, 6, 3), imagecolor.Palette{imagecolor.Transparent, imagecolor.RGBA{R: 255, A: 255}})
	frame.SetColorIndex(3, 2, 1)
	other := image.NewPaletted(image.Rect(0, 0, 8, 4), frame.Palette)
	other.SetColorIndex(0, 0, 1)
	var encoded bytes.Buffer
	if err := gif.EncodeAll(&encoded, &gif.GIF{
		Image:  []*image.Paletted{frame, other},
		Delay:  []int{1, 1},
		Config: image.Config{ColorModel: frame.Palette, Width: 8, Height: 4},
	}); err != nil {
		t.Fatal(err)
	}
	logo, err := ReadLogo(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(logo.png))
	if err != nil || decoded.Bounds() != image.Rect(0, 0, 8, 4) {
		t.Fatalf("logical screen was lost: %v, %v", decoded, err)
	}
	_, _, _, alpha := decoded.At(0, 0).RGBA()
	if alpha != 0 || imagecolor.NRGBAModel.Convert(decoded.At(3, 2)) != (imagecolor.NRGBA{R: 255, A: 255}) {
		t.Fatal("GIF frame, offset, or transparency changed")
	}
}

func TestLogoSVGUsesTransparentViewport(t *testing.T) {
	const svg = "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"60\" height=\"60\" viewBox=\"10 20 60 30\"><path fill=\"#0044aa\" fill-opacity=\"0.5\" d=\"M10 20h60v30h-60z\"/></svg>"
	logo, err := ReadLogoWithOptions(strings.NewReader("\ufeff"+svg), LogoOptions{RemoveWhiteBackground: true})
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(logo.png))
	if err != nil {
		t.Fatal(err)
	}
	if logo.width != logoRasterSide || logo.height != logoRasterSide {
		t.Fatalf("SVG viewport was lost: %d x %d", logo.width, logo.height)
	}
	_, _, _, outside := img.At(10, 10).RGBA()
	_, _, b, center := img.At(logoRasterSide/2, logoRasterSide/2).RGBA()
	if outside != 0 || center < 0x7000 || center > 0x9000 || b == 0 {
		t.Fatalf("lost transparent padding or partial alpha: outside=%x center=%x blue=%x", outside, center, b)
	}
}

func TestLogoRejectsUnsupportedAndInvalidSVG(t *testing.T) {
	for _, body := range []string{
		"<svg viewBox=\"0 0 10 10\"><image href=\"https://example.invalid/logo.png\"/></svg>",
		`<svg viewBox="0 0 10 10"><style>path {stroke-dasharray: .001 .001}</style><path fill="none" stroke="black" d="M0 0L10 10"/></svg>`,
		"<svg viewBox=\"0 0 10 10\"><script>document.write('x')</script></svg>",
		"<svg viewBox=\"0 0 10 10\"><path d=\"M 0 0\" ",
		"<svg viewBox=\"0 0 10 10\"><defs><g id=\"shape\"><use href=\"#shape\"/></g></defs><use href=\"#shape\"/></svg>",
		"<svg viewBox=\"0 0 0 10\"><rect width=\"10\" height=\"10\"/></svg>",
		`<svg width="10" height="10" viewBox="0 0 0 10"><rect width="10" height="10"/></svg>`,
		`<svg viewBox="0 0 10 10"><path d="M0 0 C 1e300 1 1 2 10 10Z"/></svg>`,
		"<!DOCTYPE svg [<!ENTITY injected SYSTEM 'file:///example'>]><svg viewBox=\"0 0 1 1\"/>",
	} {
		markup := strings.Replace(body, "<svg ", `<svg xmlns="http://www.w3.org/2000/svg" `, 1)
		if _, err := ReadLogo(strings.NewReader(markup)); err != errLogo {
			t.Errorf("accepted invalid or unsupported SVG or returned a revealing error: %v", err)
		}
	}
}

func TestLogoSVGRendersTransformsStrokesGradientsAndHoles(t *testing.T) {
	const markup = `<svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
		<defs>
		  <linearGradient id="paint" gradientUnits="userSpaceOnUse" x1="0" x2="50" y1="0" y2="0">
			<stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/>
		  </linearGradient>
		</defs>
		<g transform="scale(2)">
		  <path fill="url(#paint)" d="M0 0H50V30H0Z M15 5V25H35V5Z"/>
		  <path fill="none" stroke="#008000" stroke-width="6" d="M5 40H45"/>
		</g>
	</svg>`
	logo, err := ReadLogo(strings.NewReader(markup))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(logo.png))
	if err != nil {
		t.Fatal(err)
	}
	sample := func(x, y int) (r, g, b, a uint32) {
		return img.At(x*logo.width/100, y*logo.height/100).RGBA()
	}
	lr, _, lb, la := sample(15, 30)
	rr, _, rb, ra := sample(85, 30)
	_, _, _, hole := sample(50, 30)
	_, green, _, stroke := sample(50, 84)
	_, _, _, out := sample(50, 88)
	if lr <= lb || rb <= rr || la != 0xffff || ra != 0xffff {
		t.Fatalf("lost transformed gradient: left (%x,%x,%x), right (%x,%x,%x)", lr, lb, la, rr, rb, ra)
	}
	if hole != 0 || stroke != 0xffff || green == 0 || out != 0 {
		t.Fatalf("lost transparent outline or scaled stroke: hole=%x stroke=%x green=%x outside=%x", hole, stroke, green, out)
	}
}

func TestLogoSVGPreservesGradientOpacityAndEvenOddFill(t *testing.T) {
	const markup = `<svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
		<defs>
		  <linearGradient id="paint" gradientUnits="userSpaceOnUse" x1="0" x2="100" gradientTransform="rotate(90)">
			<stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/>
		  </linearGradient>
		</defs>
		<style>.mark { /* partial alpha */ opacity: .5; fill: url(#paint); fill-rule: evenodd }</style>
		<path class="mark" d="M0 0H100V100H0Z M30 30H70V70H30Z"/>
	</svg>`
	logo, err := ReadLogo(strings.NewReader(markup))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(logo.png))
	if err != nil {
		t.Fatal(err)
	}
	x := logo.width / 2
	topR, _, topB, topAlpha := img.At(x, logo.height/8).RGBA()
	bottomR, _, bottomB, bottomAlpha := img.At(x, 7*logo.height/8).RGBA()
	_, _, _, hole := img.At(x, logo.height/2).RGBA()
	if topR <= topB || bottomB <= bottomR || hole != 0 {
		t.Fatalf("incorrect SVG gradient transform or even-odd hole: top=(%x,%x), bottom=(%x,%x), hole=%x", topR, topB, bottomR, bottomB, hole)
	}
	for _, alpha := range []uint32{topAlpha, bottomAlpha} {
		if alpha < 0x7800 || alpha > 0x8800 {
			t.Fatalf("gradient opacity was lost: %x", alpha)
		}
	}
}

func TestLogoSVGBypassesRegisteredDecoders(t *testing.T) {
	image.RegisterFormat("logo-test-svg", "<?xml", nil, func(io.Reader) (image.Config, error) {
		panic("logo used a decoder registered by the caller")
	})
	const markup = `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10"/></svg>`
	if _, err := ReadLogo(strings.NewReader(markup)); err != nil {
		t.Fatal(err)
	}
	if _, err := decodeLogo([]byte(markup), true); err != errLogo {
		t.Fatalf("PNG-only artwork accepted SVG: %v", err)
	}
}

func TestAddImageKeepsWordRasterFormatContract(t *testing.T) {
	var buf bytes.Buffer
	if err := bmp.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if _, kind, err := image.DecodeConfig(bytes.NewReader(buf.Bytes())); kind != "bmp" || err != nil {
		t.Fatalf("invalid BMP fixture: %s, %v", kind, err)
	}
	if err := AddImage(nil, buf.Bytes(), 0, 0); err != image.ErrFormat {
		t.Fatalf("direct Word embedding should reject BMP before writing a part: %v", err)
	}
}
