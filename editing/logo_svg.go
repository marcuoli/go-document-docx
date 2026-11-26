package editing

import (
	"bytes"
	"encoding/xml"
	"image"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/xo/okage"
	"github.com/xo/okage/svg"
)

const svgNamespace = "http://www.w3.org/2000/svg"

var svgNumber = regexp.MustCompile(`[+-]?(?:[0-9]*\.[0-9]+|[0-9]+\.?)(?:[eE][+-]?[0-9]+)?`)

func decodeSVGLogo(input []byte) (raster image.Image, err error) {
	input = bytes.TrimPrefix(input, []byte{0xef, 0xbb, 0xbf})
	if !admitSVGLogo(input) {
		return nil, errLogo
	}
	// Keep the supplied document and parser errors out of diagnostics.
	defer func() {
		if recover() != nil {
			raster, err = nil, errLogo
		}
	}()
	// The renderer also has file and data image resolvers for other uses.
	// Disable both, even though the logo admission excludes image elements.
	tree, err := svg.FromData(input, svg.WithMaxDepth(32), svg.WithImageHrefResolver(svg.ImageHrefResolver{
		ResolveString: func(string) (svg.ImageKind, bool) { return nil, false },
		ResolveData:   func(string, []byte) (svg.ImageKind, bool) { return nil, false },
	}))
	if err != nil || !tree.Root().HasChildren() {
		return nil, errLogo
	}
	sw, sh := float64(tree.Size().Width()), float64(tree.Size().Height())
	if !(sw > 0 && sh > 0 && max(sw, sh) <= 1<<20) {
		return nil, errLogo
	}
	scale := logoRasterSide / max(sw, sh)
	var layerPixels float64
	if !boundedSVGTree(tree.Root(), scale, &layerPixels) {
		return nil, errLogo
	}
	w, h := max(1, int(math.Round(sw*scale))), max(1, int(math.Round(sh*scale)))
	pixmap, ok := okage.NewPixmap(uint32(w), uint32(h))
	if !ok {
		return nil, errLogo
	}
	pixels := pixmap.AsMut()
	svg.Render(tree, okage.TransformFromScale(float32(scale), float32(scale)), &pixels)
	// Okage stores premultiplied RGBA, as required by image.RGBA.
	return &image.RGBA{Pix: pixmap.Data(), Stride: 4 * w, Rect: image.Rect(0, 0, w, h)}, nil
}

func boundedSVGTree(group *svg.Group, scale float64, layerPixels *float64) bool {
	for _, node := range group.Children() {
		if box, ok := svg.NodeAbsStrokeBoundingBox(node); ok {
			for _, coord := range [...]float32{box.Left(), box.Right(), box.Top(), box.Bottom()} {
				if !(math.Abs(float64(coord)*scale) <= 1<<16) {
					return false
				}
			}
		}
		switch node := node.(type) {
		case *svg.Path:
			transform := node.AbsTransform()
			for _, point := range node.Data().Points() {
				transform.MapPoint(&point)
				if !(math.Abs(float64(point.X)*scale) <= 1<<16 && math.Abs(float64(point.Y)*scale) <= 1<<16) {
					return false
				}
			}
		case *svg.Group:
			if node.ShouldIsolate() {
				if box, ok := node.AbsLayerBoundingBox(); ok {
					*layerPixels += (4 + math.Ceil(float64(box.Width())*scale)) * (4 + math.Ceil(float64(box.Height())*scale))
					if !(*layerPixels <= 12<<20) {
						return false
					}
				}
			}
			if !boundedSVGTree(node, scale, layerPixels) {
				return false
			}
		}
	}
	return true
}

func admitSVGLogo(input []byte) bool {
	decoder := xml.NewDecoder(bytes.NewReader(input))
	depth, elements, shapes, geometryBytes, styleBytes, styleDepth := 0, 0, 0, 0, 0, 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return depth == 0 && shapes > 0
		}
		if err != nil {
			return false
		}
		switch token := token.(type) {
		case xml.StartElement:
			elements++
			depth++
			if depth > 32 || elements > 2048 || styleDepth != 0 ||
				(token.Name.Local == "svg") != (elements == 1) || (depth == 1 && elements != 1) ||
				token.Name.Space != svgNamespace {
				return false
			}
			switch token.Name.Local {
			case "rect", "circle", "ellipse", "line", "path", "polygon", "polyline":
				shapes++
			case "style":
				styleDepth = depth
			case "svg", "g", "defs", "linearGradient", "radialGradient", "stop", "title", "desc":
			default:
				// Embedded assets, reusable subtrees and filter layers can
				// amplify work independently of the output size.
				return false
			}
			for _, attr := range token.Attr {
				name := strings.ToLower(attr.Name.Local)
				if name == "href" || strings.HasPrefix(name, "on") {
					return false
				}
				switch name {
				case "style":
					styleBytes += len(attr.Value)
					if styleBytes > 8<<10 || strings.Contains(attr.Value, "@") {
						return false
					}
				case "viewbox":
					fields := strings.Fields(strings.ReplaceAll(attr.Value, ",", " "))
					if len(fields) != 4 {
						return false
					}
					for i, field := range fields {
						value, err := strconv.ParseFloat(field, 64)
						if err != nil || !(math.Abs(value) <= 1<<20) || i >= 2 && value <= 0 {
							return false
						}
					}
				case "d", "points", "transform", "gradienttransform", "stroke-width", "stroke-dasharray", "stroke-dashoffset",
					"x", "y", "x1", "x2", "y1", "y2", "cx", "cy", "fx", "fy", "r", "rx", "ry", "width", "height":
					geometryBytes += len(attr.Value)
					if geometryBytes > 64<<10 || !boundedSVGNumbers(attr.Value) {
						return false
					}
				}
			}
		case xml.EndElement:
			if depth == styleDepth {
				styleDepth = 0
			}
			depth--
		case xml.CharData:
			if depth == 0 && len(bytes.TrimSpace(token)) != 0 {
				return false
			}
			if styleDepth != 0 {
				styleBytes += len(token)
				if styleBytes > 8<<10 || bytes.Contains(token, []byte("@")) {
					return false
				}
			}
		case xml.ProcInst:
			if token.Target != "xml" || elements != 0 {
				return false
			}
		case xml.Directive:
			return false
		}
	}
}

func boundedSVGNumbers(value string) bool {
	for _, field := range svgNumber.FindAllString(value, -1) {
		n, err := strconv.ParseFloat(field, 64)
		if err != nil || !(math.Abs(n) <= 1<<20) {
			return false
		}
	}
	return true
}
