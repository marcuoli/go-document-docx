package editing

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	imagecolor "image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"

	"github.com/marcuoli/go-document-docx/report"
	"github.com/marcuoli/go-document-docx/tabledata"
)

func TestLogoFitsBothAxes(t *testing.T) {
	options := report.Defaults()
	doc, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.Table(tabledata.Data{Headers: []string{"Brand"}, Rows: [][]string{{"Caption"}}}, TableOptions{}); err != nil {
		t.Fatal(err)
	}
	var opening bytes.Buffer
	if err := doc.Write(t.Context(), &opening); err != nil {
		t.Fatal(err)
	}
	upper := image.NewNRGBA(image.Rect(0, 0, 360, 30))
	upper.SetNRGBA(0, 0, imagecolor.NRGBA{R: 12, A: 255})
	top, err := encodeLogo(upper)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name             string
		w, h             int
		rasterW, rasterH int
		stackW, stackH   int
		twipsW, twipsH   int
	}{
		{"portrait", 50, 100, 50, 100, 131, 263, 850, 1700},
		{"landscape", 100, 50, 100, 50, 263, 131, 1700, 850},
		{"square", 50, 50, 50, 50, 263, 263, 1700, 1700},
		{"downsampled", 997, 791, 640, 507, 263, 208, 1700, 1348},
		{"thin", 4096, 1, 4096, 1, 263, 1, 1700, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			pic := image.NewNRGBA(image.Rect(0, 0, test.w, test.h))
			draw.Draw(pic, pic.Bounds(), &image.Uniform{C: imagecolor.NRGBA{B: 233, A: 255}}, image.Point{}, draw.Src)
			var input bytes.Buffer
			if err := jpeg.Encode(&input, pic, nil); err != nil {
				t.Fatal(err)
			}
			logo, err := ReadLogo(&input)
			if err != nil {
				t.Fatal(err)
			}
			raster, err := png.DecodeConfig(bytes.NewReader(logo.png))
			if err != nil || raster.Width != test.rasterW || raster.Height != test.rasterH {
				t.Fatalf("wrong normalized raster size %+v: %v", raster, err)
			}
			if test.name == "square" {
				original := bytes.Clone(opening.Bytes())
				output, err := logo.CenterInTableCell(t.Context(), opening.Bytes(), options, LogoCell{Table: 1, SideTwips: 1700})
				if err == nil || output != nil || !bytes.Equal(original, opening.Bytes()) {
					t.Fatalf("invalid placement changed the document: %v", err)
				}
			}
			content, err := logo.CenterInTableCell(t.Context(), opening.Bytes(), options, LogoCell{Row: 1, SideTwips: 1700, SpaceBeforeTwips: 200})
			if err != nil {
				t.Fatal(err)
			}
			got, err := Open(t.Context(), content, options)
			if err != nil {
				t.Fatal(err)
			}
			rows := got.Native().Tables()[0].Rows()
			cell := rows[1].Cells()[0]
			paragraph := cell.Paragraphs()[1]
			images := paragraph.Images()
			if len(images) != 1 || rows[0].Cells()[0].Paragraphs()[0].Text() != "Brand" || cell.Paragraphs()[0].Text() != "Caption" {
				t.Fatal("logo placement changed the existing table")
			}
			extent := fmt.Sprintf(`<wp:extent cx="%d" cy="%d"`, test.twipsW*635, test.twipsH*635)
			if !strings.Contains(logoDocumentXML(t, content), extent) {
				t.Fatalf("missing Word extent %s", extent)
			}
			stacked, err := logo.StackBelowPNG(top, 263, 10)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := png.Decode(bytes.NewReader(stacked))
			if err != nil {
				t.Fatal(err)
			}
			if decoded.Bounds().Dy() != 40+test.stackH || decoded.Bounds().Dx() != 360 || imagecolor.NRGBAModel.Convert(decoded.At(0, 0)) != upper.NRGBAAt(0, 0) {
				t.Fatal("the composite changed the upper artwork or logo height")
			}
			var width int
			for x := 0; x < 360; x++ {
				if _, _, _, alpha := decoded.At(x, 40+test.stackH/2).RGBA(); alpha != 0 {
					width++
				}
			}
			if width != test.stackW {
				t.Fatalf("logo width = %d, want %d", width, test.stackW)
			}
		})
	}
}

func logoDocumentXML(t *testing.T, document []byte) string {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(document), int64(len(document)))
	if err != nil {
		t.Fatal(err)
	}
	part, err := archive.Open("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer part.Close()
	xml, err := io.ReadAll(part)
	if err != nil {
		t.Fatal(err)
	}
	return string(xml)
}

func TestLogoInputLimitsAndZeroValue(t *testing.T) {
	for _, input := range []string{"plain text", strings.Repeat("x", maxLogoInputBytes+1)} {
		if _, err := ReadLogo(strings.NewReader(input)); err == nil {
			t.Fatal("invalid input was accepted")
		}
	}
	var zero Logo
	original := []byte("unchanged")
	data, err := zero.CenterInTableCell(t.Context(), original, report.Defaults(), LogoCell{})
	if err != nil || !bytes.Equal(data, original) {
		t.Fatal("zero logo changed the document")
	}
	data, err = zero.StackBelowPNG(original, 0, 0)
	if err != nil || !bytes.Equal(data, original) {
		t.Fatal("zero logo changed the PNG")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := zero.CenterInTableCell(ctx, original, report.Defaults(), LogoCell{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation was lost: %v", err)
	}
}
