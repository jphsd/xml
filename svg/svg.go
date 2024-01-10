package svg

import (
	"fmt"
	g2d "github.com/jphsd/graphics2d"
	"github.com/jphsd/graphics2d/util"
	"github.com/jphsd/xml"
	stdimg "image"
	"image/color"
	"image/draw"
	"math"
)

// Draw is a convenience function to render the svg dom into an image. The svg is scaled to the image size.
func Draw(dst draw.Image, dom *xml.Element) *SVG {
	// Render the DOM
	proc := NewSVG()
	proc.Process(dom)

	// Calculate xfm to map svg bounds to dst image bounds while maintaining the aspect ratio
	renderer := proc.Rend
	dbounds := util.RectToBB(dst.Bounds())
	ddx, ddy := dbounds[1][0]-dbounds[0][0], dbounds[1][1]-dbounds[0][1]
	sbounds := util.RectToBB(renderer.Bounds())
	sdx, sdy := sbounds[1][0]-sbounds[0][0], sbounds[1][1]-sbounds[0][1]
	sx := ddx / sdx
	if sx*sdy > ddy {
		sx = ddy / sdy
	}
	xfm := g2d.Translate(-sbounds[0][0], -sbounds[0][1])
	xfm.Scale(sx, sx)
	xfm.Translate(dbounds[0][0], dbounds[0][1])
	renderer.Render(dst, xfm)
	return proc
}

// Image renders the svg dom into an empty image with the same bounds as the dom.
func Image(dom *xml.Element) (*stdimg.RGBA, *SVG) {
	proc := NewSVG()
	proc.Process(dom)

	renderer := proc.Rend
	res := renderer.Image()

	return res, proc
}

// SVG contains the current context - the image being drawn into, the style and view transforms, and the
// defined clip paths and shapes.
type SVG struct {
	Xfm  *g2d.Aff3               // Path transform
	Clip map[string]*g2d.Shape   // Clip path ids to clip shapes
	Defs map[string]*xml.Element // Element ids to elements
	Rend *g2d.Renderable         // Renderable paths and fillers
}

func NewSVG() *SVG {
	return &SVG{g2d.NewAff3(), make(map[string]*g2d.Shape), make(map[string]*xml.Element), &g2d.Renderable{}}
}

func (svg *SVG) Copy() *SVG {
	return &SVG{svg.Xfm.Copy(), svg.Clip, svg.Defs, svg.Rend}
}

func (svg *SVG) Process(elt *xml.Element) {
	if elt.Type != xml.Node || elt.Attributes["display"] == "none" {
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
	case "defs":
		svg.DefsElt(elt)
	case "use":
		svg.UseElt(elt)
	case "clipPath":
		svg.ClipPathElt(elt)
	default:
		fmt.Printf("%s not implemented\n", name)
	}
}

// Element functions

func (svg *SVG) SVGElt(elt *xml.Element) {
	orig := svg.Xfm.Copy()

	// Set/Capture initial fill and style
	_, ok := elt.Attributes["fill"]
	if !ok {
		elt.Attributes["fill"] = "#000"
	}
	_, ok = elt.Attributes["fill-opacity"]
	if !ok {
		elt.Attributes["fill-opacity"] = "1"
	}
	_, ok = elt.Attributes["stroke"]
	if !ok {
		elt.Attributes["stroke"] = "none"
	}
	_, ok = elt.Attributes["stroke-opacity"]
	if !ok {
		elt.Attributes["stroke-opacity"] = "1"
	}
	_, ok = elt.Attributes["stroke-linecap"]
	if !ok {
		elt.Attributes["stroke-linecap"] = "butt"
	}
	_, ok = elt.Attributes["stroke-linejoin"]
	if !ok {
		elt.Attributes["stroke-linejoin"] = "miter"
	}
	_, ok = elt.Attributes["stroke-miterlimit"]
	if !ok {
		elt.Attributes["stroke-miterlimit"] = "4"
	}
	_, ok = elt.Attributes["clip-path"]
	if !ok {
		elt.Attributes["clip-path"] = ""
	}

	// Process all children
	for _, elt := range elt.Children {
		svg.Process(elt)
	}

	svg.Xfm = orig
}

func (svg *SVG) GroupElt(elt *xml.Element) {
	nsvg := svg.Copy()
	xfm := svg.Transform(elt)
	if xfm != nil {
		nsvg.Xfm.Concatenate(*xfm)
	}
	for _, child := range elt.Children {
		nsvg.Process(child)
	}
}

func (svg *SVG) PathElt(elt *xml.Element) {
	paths := PathsFromDescription(elt.Attributes["d"])

	// Can't use renderPath since there might be multiple paths
	xfm := svg.Transform(elt)
	if xfm != nil {
		svg.Xfm.Concatenate(*xfm)
	}

	shape := g2d.NewShape(paths...)
	shape = shape.Transform(svg.Xfm)

	inside, id := insideClipPath(elt)
	if inside {
		svg.Clip[id].AddShapes(shape)
		return
	}

	fill, pen := svg.FillStroke(elt, shape.BoundingBox())

	cid := ParseUrlId(elt.Attributes["clip-path"])
	clip := svg.Clip[cid]
	if fill != nil {
		svg.Rend.AddClippedShape(shape, clip, fill.Filler, nil)
	}

	if pen != nil {
		svg.Rend.AddClippedPennedShape(shape, clip, pen, nil)
	}
}

func (svg *SVG) RectElt(elt *xml.Element) {
	x1 := ParseValue(elt.Attributes["x"])
	y1 := ParseValue(elt.Attributes["y"])
	w := ParseValue(elt.Attributes["width"])
	h := ParseValue(elt.Attributes["height"])
	x2, y2 := x1+w, y1+h

	// Handle optional rx, ry
	var rxp, ryp bool
	rxa, rxp := elt.Attributes["rx"]
	rya, ryp := elt.Attributes["ry"]

	var path *g2d.Path
	if rxp || ryp {
		rx := ParseValue(rxa)
		ry := ParseValue(rya)
		if rxp && !ryp {
			ry = rx
		} else if ryp && !rxp {
			rx = ry
		}
		if rx < 0 {
			rx = 0
		} else if rx > w/2 {
			rx = w / 2
		}
		if ry < 0 {
			ry = 0
		} else if ry > h/2 {
			ry = h / 2
		}
		x3, x4 := x1+rx, x2-rx
		y3, y4 := y1+ry, y2-ry
		path = g2d.Line([]float64{x3, y1}, []float64{x4, y1})
		path.Concatenate(g2d.EllipticalArc([]float64{x4, y3}, rx, ry, -math.Pi/2, math.Pi/2, 0, g2d.ArcOpen))
		path.Concatenate(g2d.Line([]float64{x2, y3}, []float64{x2, y4}))
		path.Concatenate(g2d.EllipticalArc([]float64{x4, y4}, rx, ry, 0, math.Pi/2, 0, g2d.ArcOpen))
		path.Concatenate(g2d.Line([]float64{x4, y2}, []float64{x3, y2}))
		path.Concatenate(g2d.EllipticalArc([]float64{x3, y4}, rx, ry, math.Pi/2, math.Pi/2, 0, g2d.ArcOpen))
		path.Concatenate(g2d.Line([]float64{x1, y4}, []float64{x1, y3}))
		path.Concatenate(g2d.EllipticalArc([]float64{x3, y3}, rx, ry, math.Pi, math.Pi/2, 0, g2d.ArcOpen))
		path.Close()
	} else {
		path = g2d.Polygon([]float64{x1, y1}, []float64{x2, y1}, []float64{x2, y2}, []float64{x1, y2})
	}

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

func (svg *SVG) DefsElt(elt *xml.Element) {
	for _, child := range elt.Children {
		if child.Type == xml.Node {
			id, ok := child.Attributes["id"]
			if ok {
				svg.Defs[id] = child
			}
		}
	}
}

func (svg *SVG) UseElt(elt *xml.Element) {
	nsvg := svg.Copy()

	// Find href
	id, ok := elt.Attributes["href"]
	if !ok {
		fmt.Println("no id attribute in <use>")
		return
	}
	id = id[1:]
	delt, ok := svg.Defs[id]
	if !ok {
		fmt.Printf("id %s not defined\n", id)
		return
	}

	// Process any transform
	xfm := svg.Transform(elt)
	if xfm != nil {
		nsvg.Xfm.Concatenate(*xfm)
	}
	// Add <use> x, y translation
	x := ParseValue(elt.Attributes["x"])
	y := ParseValue(elt.Attributes["y"])
	nsvg.Xfm.Concatenate(*g2d.Translate(x, y))

	// Clone and attach current as parent
	clone := delt.Copy()
	clone.Parent = elt
	nsvg.Process(clone)
}

func (svg *SVG) ClipPathElt(elt *xml.Element) {
	// Raw paths are all that matter - style, fill, stroke etc are all ignored
	// Operates in the current user coordinate system
	// All paths are or'd together
	// clip-path can be specified (intersection of the two) - ignore since that would yield an image and not a shape

	id, ok := elt.Attributes["id"]
	if !ok {
		return
	}

	nsvg := svg.Copy()
	xfm := svg.Transform(elt)
	if xfm != nil {
		nsvg.Xfm.Concatenate(*xfm)
	}

	svg.Clip[id] = &g2d.Shape{}
	for _, child := range elt.Children {
		nsvg.Process(child)
	}
}

// End of Element functions

func (svg *SVG) Transform(elt *xml.Element) *g2d.Aff3 {
	return ParseTransform(elt.Attributes["transform"])
}

func (svg *SVG) FillStroke(elt *xml.Element, bb [][]float64) (*g2d.Pen, *g2d.Pen) {
	var fill, pen *g2d.Pen

	//fmt.Printf("\n<%s>\n", elt.Name.Local)
	//for k, v := range elt.Attributes {
	//	fmt.Printf("%s: %s\n", k, v)
	//}

	// visibility
	if elt.Attributes["visibility"] == "hidden" {
		return fill, pen
	}

	// fill and fill-opacity
	col := ParseColor(elt.Attributes["fill"])
	if col != nil {
		fcol, _ := col.(color.RGBA)
		fop := ParseValue(elt.Attributes["fill-opacity"])
		if fop < 0 {
			fop = 0
		} else if fop > 1 {
			fop = 1
		}
		if fop < 1 {
			// RGBA is premultiplied
			r, g, b, a := float64(fcol.R)*fop, float64(fcol.G)*fop, float64(fcol.B)*fop, 0xff*fop
			fcol.R, fcol.G, fcol.B, fcol.A = uint8(r), uint8(g), uint8(b), uint8(a)
		}
		fill = g2d.NewPen(fcol, 1)
	}

	// stroke and stroke-opacity
	col = ParseColor(elt.Attributes["stroke"])
	if col == nil {
		return fill, pen
	}
	scol, _ := col.(color.RGBA)
	sop := ParseValue(elt.Attributes["stroke-opacity"])
	if sop < 0 {
		sop = 0
	} else if sop > 1 {
		sop = 1
	}
	if sop < 1 {
		// RGBA is premultiplied
		r, g, b, a := float64(scol.R)*sop, float64(scol.G)*sop, float64(scol.B)*sop, 0xff*sop
		scol.R, scol.G, scol.B, scol.A = uint8(r), uint8(g), uint8(b), uint8(a)
	}

	// stroke-width
	sw, _ := ParseValueUnit(elt.Attributes["stroke-width"])
	if util.Equals(sw, 0) {
		sw = 1
	}

	// vector-effect attribute is from SVG12
	if elt.Attributes["vector-effect"] != "non-scaling-stroke" {
		// Per SVG spec sw is scaled by the current xfm
		// Calc sx and sy by transforming points sw in x and y away from
		// the shape's minimum and then combining them
		pt0 := bb[0] // Min
		pt1 := []float64{pt0[0] + sw, pt0[1]}
		pt2 := []float64{pt0[0], pt0[1] + sw}
		pts := svg.Xfm.Apply(pt0, pt1, pt2)
		dx, dy := pts[1][0]-pts[0][0], pts[1][1]-pts[0][1]
		sx := math.Hypot(dx, dy)
		dx, dy = pts[2][0]-pts[0][0], pts[2][1]-pts[0][1]
		sy := math.Hypot(dx, dy)
		sw = math.Sqrt(sx * sy) // geometric mean
	}

	pen = g2d.NewPen(scol, sw)

	// stroke-linecap: butt, [round, square]
	attr, ok := elt.Attributes["stroke-linecap"]
	if ok {
		tsp, _ := pen.Stroke.(*g2d.StrokeProc)
		switch attr {
		case "round":
			tsp.CapStartFunc = g2d.CapRound
			tsp.CapEndFunc = g2d.CapRound
			tsp.PointFunc = g2d.PointCircle
		case "square":
			tsp.CapStartFunc = g2d.CapSquare
			tsp.CapEndFunc = g2d.CapSquare
			tsp.PointFunc = g2d.PointSquare
		default:
			fallthrough
		case "butt":
			tsp.CapStartFunc = g2d.CapButt
			tsp.CapEndFunc = g2d.CapButt
			tsp.PointFunc = g2d.PointCircle
		}
	}

	// stroke-linejoin: miter, [round, bevel]
	// stroke-miterlimit: 4 [1,) ratio of miter length to stroke width
	attr, ok = elt.Attributes["stroke-linejoin"]
	if ok {
		tsp, _ := pen.Stroke.(*g2d.StrokeProc)
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

	return fill, pen
}

func (svg *SVG) renderPath(path *g2d.Path, elt *xml.Element) {
	xfm := svg.Transform(elt)
	if xfm != nil {
		svg.Xfm.Concatenate(*xfm)
	}

	shape := g2d.NewShape(path)
	shape = shape.Transform(svg.Xfm)

	inside, id := insideClipPath(elt)
	if inside {
		svg.Clip[id].AddShapes(shape)
		return
	}

	fill, pen := svg.FillStroke(elt, shape.BoundingBox())

	cid := ParseUrlId(elt.Attributes["clip-path"])
	clip := svg.Clip[cid]
	if fill != nil {
		svg.Rend.AddClippedShape(shape, clip, fill.Filler, nil)
	}

	if pen != nil {
		svg.Rend.AddClippedPennedShape(shape, clip, pen, nil)
	}
}

func inheritAttributes(elt *xml.Element) {
	// style stomps on presentation attributes
	ParseStyle(elt.Attributes["style"], elt.Attributes)

	if elt.Parent == nil {
		return
	}

	preserve := []string{
		"clip-path",
		"fill",
		"fill-opacity",
		"stroke",
		"stroke-opacity",
		"stroke-width",
		"stroke-linecap",
		"stroke-linejoin",
		"stroke-miterlimit",
	}

	// Make a new map with the preserved elements from the parent and then update with the child
	cattrs := elt.Attributes
	pattrs := elt.Parent.Attributes
	attrs := make(map[string]string)
	for _, attr := range preserve {
		v, ok := pattrs[attr]
		if ok {
			attrs[attr] = v
		}
	}
	for k, v := range cattrs {
		attrs[k] = v
	}
	elt.Attributes = attrs
}

func insideClipPath(elt *xml.Element) (bool, string) {
	for elt != nil {
		elt = elt.Parent
		if elt != nil && elt.Name.Local == "clipPath" {
			return true, elt.Attributes["id"]
		}
	}
	return false, ""
}
