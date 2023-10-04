package svg

import (
	"fmt"
	g2d "github.com/jphsd/graphics2d"
	"github.com/jphsd/graphics2d/util"
	"github.com/jphsd/xml"
	"image"
	"image/draw"
	"math"
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

	inheritAttributes(elt)

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

	// Capture initial fill and style
	svg.Fill, svg.Pen = svg.FillStroke(elt)

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
	nsvg.Fill, nsvg.Pen = svg.FillStroke(elt)
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
	ParseStyle(elt.Attributes["style"], elt.Attributes)

	// fill: black
	attr := elt.Attributes["fill"]
	if attr != "" {
		fcol := ParseColor(attr)
		if fcol != nil {
			fill = g2d.NewPen(fcol, 1)
		} else {
			fill = nil
		}
	}

	// stroke: none
	scol := ParseColor(elt.Attributes["stroke"])
	if scol == nil && pen == nil {
		return fill, pen
	}

	// stroke-width: 1
	sw, _ := ParseValueUnit(elt.Attributes["stroke-width"])
	if util.Equals(sw, 0) {
		sw = 1
	}

	var npen *g2d.Pen
	if pen == nil {
		// Create new pen with miter line join and butt caps
		pw := sw * svg.PenS / 2
		ang := 2 * math.Asin(0.25) // miter limit = 4
		mj := &g2d.MiterJoin{ang, g2d.JoinBevel}
		sp := &g2d.StrokeProc{
			&g2d.TraceProc{pw, 0.5, mj.JoinMiter},
			&g2d.TraceProc{-pw, 0.5, mj.JoinMiter},
			nil,
			g2d.PointCircle,
			g2d.CapButt,
			nil,
			nil}
		npen = &g2d.Pen{image.NewUniform(scol), sp, nil}
	} else {
		// Clone curent pen
		tsp, _ := pen.Stroke.(*g2d.StrokeProc)
		sp := &g2d.StrokeProc{
			tsp.RTraceProc,
			tsp.LTraceProc,
			nil,
			tsp.PointFunc,
			tsp.CapFunc,
			nil,
			nil}
		npen = &g2d.Pen{pen.Filler, sp, nil}
	}

	// stroke-linecap: butt, [round, square]
	attr = elt.Attributes["stroke-linecap"]
	if attr != "" {
		tsp, _ := npen.Stroke.(*g2d.StrokeProc)
		switch attr {
		case "round":
			tsp.CapFunc = g2d.CapRound
			tsp.PointFunc = g2d.PointCircle
		case "square":
			tsp.CapFunc = g2d.CapSquare
			tsp.PointFunc = g2d.PointSquare
		default:
			fallthrough
		case "butt":
			tsp.CapFunc = g2d.CapButt
			tsp.PointFunc = g2d.PointCircle
		}
	}

	// stroke-linejoin: miter, [round, bevel]
	// stroke-miterlimit: 4 [1,) ratio of miter length to stroke width
	attr = elt.Attributes["stroke-linejoin"]
	if attr != "" {
		tsp, _ := npen.Stroke.(*g2d.StrokeProc)
		switch attr {
		default:
			fallthrough
		case "miter":
			ml := 4.0
			attr = elt.Attributes["stroke-miterlimit"]
			if attr != "" {
				ml = ParseValue(attr)
				if ml < 1 {
					ml = 1
				}
			}
			mj := &g2d.MiterJoin{2 * math.Asin(1/ml), g2d.JoinBevel}
			tsp.RTraceProc.JoinFunc = mj.JoinMiter
			tsp.LTraceProc.JoinFunc = mj.JoinMiter
		case "round":
			tsp.RTraceProc.JoinFunc = g2d.JoinRound
			tsp.LTraceProc.JoinFunc = g2d.JoinRound
		case "bevel":
			tsp.RTraceProc.JoinFunc = g2d.JoinBevel
			tsp.LTraceProc.JoinFunc = g2d.JoinBevel
		}
	}

	return fill, npen
}

func (svg *SVG) renderPath(path *g2d.Path, elt *xml.Element) {
	nsvg := svg.Copy()
	xfm := svg.Transform(elt)
	if xfm != nil {
		nsvg.Pxfm.Concatenate(*xfm)
	}
	nsvg.Fill, nsvg.Pen = svg.FillStroke(elt)
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

func inheritAttributes(elt *xml.Element) {
	if elt.Parent == nil {
		return
	}

	// Assumes ParseStyle has already been called on parent
	preserve := []string{
		"fill",
		"stroke",
		"stroke-linewidth",
		"stroke-linecap",
		"stroke-linejoin",
		"stroke-miterlimit",
	}

	// Make a new map with the preserved elements from the parent and then update with the child
	cattrs := elt.Attributes
	pattrs := elt.Parent.Attributes
	attrs := make(map[string]string)
	for _, attr := range preserve {
		attrs[attr] = pattrs[attr]
	}
	for k, v := range cattrs {
		attrs[k] = v
	}
	elt.Attributes = attrs
}
