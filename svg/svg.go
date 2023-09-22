package svg

import (
	"fmt"
	g2d "github.com/jphsd/graphics2d"
	"github.com/jphsd/graphics2d/color"
	"github.com/jphsd/graphics2d/image"
	"github.com/jphsd/graphics2d/util"
	"github.com/jphsd/xml"
	"image/draw"
)

// Draw renders the SVG data contained in dom into the destination image.
func Draw(dst draw.Image, dom *xml.Element) {
	// Process dom into dst
}

// TODO remove this once tested
var N int

type SVG struct {
	Img  draw.Image
	Fill *g2d.Pen
	Pen  *g2d.Pen
	Xfm  *g2d.Aff3
}

func NewSVG(dst draw.Image) *SVG {
	// SVG default for fill is black, and for stroke is none
	N = -1
	return &SVG{dst, g2d.BlackPen, nil, g2d.NewAff3()}
}

func (svg *SVG) Copy() *SVG {
	return &SVG{svg.Img, svg.Fill, svg.Pen, svg.Xfm.Copy()}
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
	// Adjust xfm for viewBox (maintain aspect ratio)
	// TODO - viewBox
	// Process all children
	for _, elt := range elt.Children {
		svg.Process(elt)
	}
}

func (svg *SVG) GroupElt(elt *xml.Element) {
	nsvg := svg.Copy()
	// Pick up any transform, stroke or fill settings before walking children
	// TODO - transform
	nsvg.Fill, nsvg.Pen = svg.FillStroke(elt)
	for _, elt := range elt.Children {
		nsvg.Process(elt)
	}
}

func (svg *SVG) PathElt(elt *xml.Element) {
	paths := PathsFromDescription(elt.Attributes["d"])
	fill, stroke := svg.FillStroke(elt)
	// TODO - pick up xfm
	svg.renderShape(g2d.NewShape(paths...), fill, stroke)
}

func (svg *SVG) RectElt(elt *xml.Element) {
	x := ParseValue(elt.Attributes["x"])
	y := ParseValue(elt.Attributes["y"])
	w := ParseValue(elt.Attributes["width"])
	h := ParseValue(elt.Attributes["height"])
	path := g2d.Polygon([]float64{x, y}, []float64{x + w, y}, []float64{x + w, y + h}, []float64{x, y + h})
	fill, stroke := svg.FillStroke(elt)
	// TODO - pick up xfm
	svg.renderShape(g2d.NewShape(path), fill, stroke)
}

func (svg *SVG) CircleElt(elt *xml.Element) {
	cx := ParseValue(elt.Attributes["cx"])
	cy := ParseValue(elt.Attributes["cy"])
	r := ParseValue(elt.Attributes["r"])
	path := g2d.Circle([]float64{cx, cy}, r)
	fill, stroke := svg.FillStroke(elt)
	// TODO - pick up xfm
	svg.renderShape(g2d.NewShape(path), fill, stroke)
}

func (svg *SVG) EllipseElt(elt *xml.Element) {
	cx := ParseValue(elt.Attributes["cx"])
	cy := ParseValue(elt.Attributes["cy"])
	rx := ParseValue(elt.Attributes["rx"])
	ry := ParseValue(elt.Attributes["ry"])
	path := g2d.Ellipse([]float64{cx, cy}, rx, ry, 0)
	fill, stroke := svg.FillStroke(elt)
	// TODO - pick up xfm
	svg.renderShape(g2d.NewShape(path), fill, stroke)
}

func (svg *SVG) LineElt(elt *xml.Element) {
	x1 := ParseValue(elt.Attributes["x1"])
	y1 := ParseValue(elt.Attributes["y1"])
	x2 := ParseValue(elt.Attributes["x2"])
	y2 := ParseValue(elt.Attributes["y2"])
	path := g2d.Line([]float64{x1, y1}, []float64{x2, y2})
	fill, stroke := svg.FillStroke(elt)
	// TODO - pick up xfm
	svg.renderShape(g2d.NewShape(path), fill, stroke)
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
	fill, stroke := svg.FillStroke(elt)
	// TODO - pick up xfm
	svg.renderShape(g2d.NewShape(path), fill, stroke)
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
	fill, stroke := svg.FillStroke(elt)
	// TODO - pick up xfm
	svg.renderShape(g2d.NewShape(path), fill, stroke)
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
			sw = 1 // SVG default stroke width (JH - add to SVG?)
		}
		if scol != nil {
			pen = g2d.NewPen(scol, sw)
		} else {
			pen = nil
		}
	}

	return fill, pen
}

func (svg *SVG) renderShape(shape *g2d.Shape, fill, stroke *g2d.Pen) {
	// TODO - apply xfm
	if N == -1 {
		if fill != nil {
			g2d.FillShape(svg.Img, shape, fill)
		}
		if stroke != nil {
			g2d.DrawShape(svg.Img, shape, stroke)
		}
		return
	}
	img := image.NewRGBA(1000, 1000, color.White)
	if fill != nil {
		g2d.FillShape(img, shape, fill)
	}
	if stroke != nil {
		g2d.DrawShape(img, shape, stroke)
	}
	image.SaveImage(img, fmt.Sprintf("path-%d", N))
	N++
}
