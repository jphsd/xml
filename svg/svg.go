package svg

import (
	"fmt"
	g2d "github.com/jphsd/graphics2d"
	"github.com/jphsd/graphics2d/util"
	"github.com/jphsd/xml"
	"image/draw"
)

// Draw renders the SVG data contained in dom into the destination image.
func Draw(dst draw.Image, dom *xml.Element) {
	NewSVG(dst).Process(dom)
}

// SVG contains the current context - the imgae being drawn into, the style and veiw transforms.
type SVG struct {
	Img  draw.Image
	Fill *g2d.Pen
	Pen  *g2d.Pen
	Xfm  *g2d.Aff3
	Pxfm *g2d.Aff3
	PenS float64
}

func NewSVG(dst draw.Image) *SVG {
	// SVG default for fill is black, and for stroke is none
	return &SVG{dst, g2d.BlackPen, nil, g2d.NewAff3(), g2d.NewAff3(), 1}
}

func (svg *SVG) Copy() *SVG {
	return &SVG{svg.Img, svg.Fill, svg.Pen, svg.Xfm.Copy(), svg.Pxfm.Copy(), svg.PenS}
}

func (svg *SVG) Process(elt *xml.Element) {
	if elt.Type != xml.Node {
		return
	}

	// Process is not SVG DOM aware - ie no checking on element validity
	// Look at element and call appropriate function
	name := elt.Name.Local
	switch name {
	case "svg":
		svg.SVGElt(elt)
	case "g":
		svg.GroupElt(elt)
	case "path":
		svg.PathElt(elt)
	case "rect":
		svg.RectElt(elt)
	case "circle":
		svg.CircleElt(elt)
	case "ellipse":
		svg.EllipseElt(elt)
	case "line":
		svg.LineElt(elt)
	case "polyline":
		svg.PolylineElt(elt)
	case "polygon":
		svg.PolygonElt(elt)
	default:
		fmt.Printf("%s not implemented\n", name)
	}
}

func (svg *SVG) SVGElt(elt *xml.Element) {
	orig := svg.Xfm

	// Adjust xfm for viewBox (maintain aspect ratio)
	attr := elt.Attributes["viewBox"]
	if attr != "" {
		// Fit the viewBox to the image maintaining the vb aspect ratio
		bounds := ParseViewBox(attr)
		vbdx, vbdy := bounds[1][0]-bounds[0][0], bounds[1][1]-bounds[0][1]
		rect := svg.Img.Bounds()
		dims := [][]float64{{float64(rect.Min.X), float64(rect.Min.Y)}, {float64(rect.Max.X), float64(rect.Max.Y)}}
		imgdx, imgdy := dims[1][0]-dims[0][0], dims[1][1]-dims[0][1]
		sx, sy := imgdx/vbdx, imgdy/vbdy
		if sy < sx {
			sx = sy
		}
		//xfm := g2d.Scale(sx, sx)
		//xfm.Translate(bounds[0][0], bounds[0][1])
		xfm := g2d.Translate(bounds[0][0], bounds[0][1])
		xfm.Scale(sx, sx)
		svg.Xfm = xfm
		svg.PenS = sx
	}

	// Process all children
	for _, elt := range elt.Children {
		svg.Process(elt)
	}

	// Restore previous viewBox transform
	svg.Xfm = orig
}

func (svg *SVG) GroupElt(elt *xml.Element) {
	nsvg := svg.Copy()
	xfm := svg.Transform(elt)
	if xfm != nil {
		nsvg.Pxfm.Concatenate(*xfm)
	}
	nsvg.Fill, nsvg.Pen = svg.FillStroke(elt)
	for _, elt := range elt.Children {
		nsvg.Process(elt)
	}
}

func (svg *SVG) PathElt(elt *xml.Element) {
	paths := PathsFromDescription(elt.Attributes["d"])

	nsvg := svg.Copy()
	xfm := svg.Transform(elt)
	if xfm != nil {
		nsvg.Pxfm.Concatenate(*xfm)
	}
	fill, stroke := svg.FillStroke(elt)
	nsvg.Fill, nsvg.Pen = fill, stroke
	shape := g2d.NewShape(paths...)
	shape = shape.Transform(nsvg.Pxfm)
	nsvg.renderShape(shape)
}

func (svg *SVG) RectElt(elt *xml.Element) {
	x := ParseValue(elt.Attributes["x"])
	y := ParseValue(elt.Attributes["y"])
	w := ParseValue(elt.Attributes["width"])
	h := ParseValue(elt.Attributes["height"])
	// rx, ry
	path := g2d.Polygon([]float64{x, y}, []float64{x + w, y}, []float64{x + w, y + h}, []float64{x, y + h})

	svg.renderPath(path, elt)
}

func (svg *SVG) CircleElt(elt *xml.Element) {
	cx := ParseValue(elt.Attributes["cx"])
	cy := ParseValue(elt.Attributes["cy"])
	r := ParseValue(elt.Attributes["r"])
	path := g2d.Circle([]float64{cx, cy}, r)

	svg.renderPath(path, elt)
}

func (svg *SVG) EllipseElt(elt *xml.Element) {
	cx := ParseValue(elt.Attributes["cx"])
	cy := ParseValue(elt.Attributes["cy"])
	rx := ParseValue(elt.Attributes["rx"])
	ry := ParseValue(elt.Attributes["ry"])
	path := g2d.Ellipse([]float64{cx, cy}, rx, ry, 0)

	svg.renderPath(path, elt)
}

func (svg *SVG) LineElt(elt *xml.Element) {
	x1 := ParseValue(elt.Attributes["x1"])
	y1 := ParseValue(elt.Attributes["y1"])
	x2 := ParseValue(elt.Attributes["x2"])
	y2 := ParseValue(elt.Attributes["y2"])
	path := g2d.Line([]float64{x1, y1}, []float64{x2, y2})

	svg.renderPath(path, elt)
}

func (svg *SVG) PolylineElt(elt *xml.Element) {
	pstr := elt.Attributes["points"]
	pstr = wscpat.ReplaceAllString(pstr, " ")
	pstr = "X" + cpat.ReplaceAllString(pstr, "$1 -") // Add dummy command
	_, coords := commandCoords(pstr)
	path := g2d.NewPath([]float64{coords[0], coords[1]})
	for i := 2; i < len(coords); i += 2 {
		path.AddStep([]float64{coords[i], coords[i+1]})
	}

	svg.renderPath(path, elt)
}

func (svg *SVG) PolygonElt(elt *xml.Element) {
	pstr := elt.Attributes["points"]
	pstr = wscpat.ReplaceAllString(pstr, " ")
	pstr = "X" + cpat.ReplaceAllString(pstr, "$1 -") // Add dummy command
	_, coords := commandCoords(pstr)
	path := g2d.NewPath([]float64{coords[0], coords[1]})
	for i := 2; i < len(coords); i += 2 {
		path.AddStep([]float64{coords[i], coords[i+1]})
	}
	path.Close()

	svg.renderPath(path, elt)
}

func (svg *SVG) Transform(elt *xml.Element) *g2d.Aff3 {
	attr := elt.Attributes["transform"]
	if attr != "" {
		return ParseTransform(attr)
	}
	return nil
}

func (svg *SVG) FillStroke(elt *xml.Element) (*g2d.Pen, *g2d.Pen) {
	fill := svg.Fill
	pen := svg.Pen

	// style stomps on presentation attributes
	attr := elt.Attributes["style"]
	if attr != "" {
		ParseStyle(attr, elt.Attributes)
	}

	attr = elt.Attributes["fill"]
	if attr != "" {
		fcol := ParseColor(attr)
		if fcol != nil {
			fill = g2d.NewPen(fcol, 1)
		} else {
			fill = nil
		}
	}

	attr = elt.Attributes["stroke"]
	if attr != "" {
		scol := ParseColor(attr)
		// Check for stroke-width
		sw, _ := ParseValueUnit(elt.Attributes["stroke-width"])
		if util.Equals(sw, 0) {
			sw = 1 // SVG default stroke width (JH - add to SVG struct?)
		}
		if scol != nil {
			// Pen width is scaled by viewBox to image scale
			pen = g2d.NewPen(scol, sw*svg.PenS)
		} else {
			pen = nil
		}
	} else {
		attr = elt.Attributes["stroke-width"]
		if attr != "" {
			sw, _ := ParseValueUnit(attr)
			if util.Equals(sw, 0) {
				sw = 1 // SVG default stroke width (JH - add to SVG struct?)
			}
			pen.Stroke = g2d.NewStrokeProc(sw * svg.PenS) // Rude
		}
	}

	return fill, pen
}

func (svg *SVG) renderPath(path *g2d.Path, elt *xml.Element) {
	nsvg := svg.Copy()
	xfm := svg.Transform(elt)
	if xfm != nil {
		nsvg.Pxfm.Concatenate(*xfm)
	}
	fill, stroke := svg.FillStroke(elt)
	nsvg.Fill, nsvg.Pen = fill, stroke
	shape := g2d.NewShape(path)
	shape = shape.Transform(nsvg.Pxfm)
	nsvg.renderShape(shape)
}

func (svg *SVG) renderShape(shape *g2d.Shape) {
	// Apply viewBox xfm
	shape = shape.Transform(svg.Xfm)

	if svg.Fill != nil {
		g2d.FillShape(svg.Img, shape, svg.Fill)
	}
	if svg.Pen != nil {
		g2d.DrawShape(svg.Img, shape, svg.Pen)
	}
	return
}
