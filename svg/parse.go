package svg

import (
	"fmt"
	g2d "github.com/jphsd/graphics2d"
	"github.com/jphsd/graphics2d/color"
	"github.com/jphsd/graphics2d/util"
	stdcol "image/color"
	"math"
	"regexp"
	"strings"
)

var (
	wspat  = regexp.MustCompile(`[ \n\r\t]+`)    // Whitespace pattern
	wscpat = regexp.MustCompile(`[, \n\r\t]+`)   // Whitespace and comma pattern
	cpat   = regexp.MustCompile(`(\d)-`)         // Digit followed by -ve sign pattern (SVG allows + too...)
	lcapat = regexp.MustCompile(`([a-z]+)`)      // Lower case word pattern
	adjpat = regexp.MustCompile(`(\.\d+)(\.\d)`) // Adjacent decimals pattern
)

func PathsFromDescription(desc string) []*g2d.Path {
	cmds := commands(desc)
	cx, cy := 0.0, 0.0
	cx0, cy0 := 0.0, 0.0
	res := []*g2d.Path{}
	var path *g2d.Path
	var cp, qp []float64
	first := true
	newpath := false
	for _, cmd := range cmds {
		c, coords := commandCoords(cmd)
		switch c {
		case 'M': // MoveTo
			if path != nil && !newpath {
				res = append(res, path)
			}
			newpath = false
			for i := 0; i < len(coords); i += 2 {
				cx, cy = coords[i], coords[i+1]
				if i == 0 {
					path = g2d.NewPath([]float64{cx, cy})
					cx0, cy0 = cx, cy
				} else {
					// Additional pairs treated as L
					path.AddStep([]float64{cx, cy})
				}
			}
			qp, cp = nil, nil
		case 'm':
			if path != nil && !newpath {
				res = append(res, path)
			}
			newpath = false
			for i := 0; i < len(coords); i += 2 {
				if i == 0 {
					if first {
						// Per SVG standard
						cx, cy = coords[i], coords[i+1]
					} else {
						cx, cy = cx+coords[i], cy+coords[i+1]
					}
					path = g2d.NewPath([]float64{cx, cy})
					cx0, cy0 = cx, cy
				} else {
					// Additional pairs treated as l
					cx, cy = cx+coords[i], cy+coords[i+1]
					path.AddStep([]float64{cx, cy})
				}
			}
			qp, cp = nil, nil
		case 'L': // LineTo
			for i := 0; i < len(coords); i += 2 {
				cx, cy = coords[i], coords[i+1]
				path.AddStep([]float64{cx, cy})
			}
			qp, cp = nil, nil
		case 'l':
			for i := 0; i < len(coords); i += 2 {
				cx, cy = cx+coords[i], cy+coords[i+1]
				path.AddStep([]float64{cx, cy})
			}
			qp, cp = nil, nil
		case 'H': // HorizontalTo
			for _, v := range coords {
				cx = v
				path.AddStep([]float64{cx, cy})
			}
			qp, cp = nil, nil
		case 'h':
			for _, v := range coords {
				cx = cx + v
				path.AddStep([]float64{cx, cy})
			}
			qp, cp = nil, nil
		case 'V': // VerticalTo
			for _, v := range coords {
				cy = v
				path.AddStep([]float64{cx, cy})
			}
			qp, cp = nil, nil
		case 'v':
			for _, v := range coords {
				cy = cy + v
				path.AddStep([]float64{cx, cy})
			}
			qp, cp = nil, nil
		case 'Q': // QuadTo
			for i := 0; i < len(coords); i += 4 {
				p1 := []float64{coords[i], coords[i+1]}
				qp = p1
				p2 := []float64{coords[i+2], coords[i+3]}
				cx, cy = p2[0], p2[1]
				path.AddStep(p1, p2)
			}
			cp = nil
		case 'q':
			for i := 0; i < len(coords); i += 4 {
				cxp, cyp := cx+coords[i], cy+coords[i+1]
				p1 := []float64{cxp, cyp}
				qp = p1
				cx, cy = cx+coords[i+2], cy+coords[i+3]
				p2 := []float64{cx, cy}
				path.AddStep(p1, p2)
			}
			cp = nil
		case 'T': // SmoothQuadTo
			for i := 0; i < len(coords); i += 2 {
				// Infer p1 from reflected penultimate value of previous C/S step, else use current
				var p1 []float64
				if qp == nil {
					p1 = []float64{cx, cy}
				} else {
					dx, dy := cx-qp[0], cy-qp[1]
					p1 = []float64{cx + dx, cy + dy}
				}
				qp = p1
				p2 := []float64{coords[i], coords[i+1]}
				cx, cy = p2[0], p2[1]
				path.AddStep(p1, p2)
			}
			cp = nil
		case 't':
			for i := 0; i < len(coords); i += 2 {
				// Infer p1 from reflected penultimate value of previous C/S step, else use current
				var p1 []float64
				if qp == nil {
					p1 = []float64{cx, cy}
				} else {
					dx, dy := cx-qp[0], cy-qp[1]
					p1 = []float64{cx + dx, cy + dy}
				}
				qp = p1
				cx, cy = cx+coords[i], cy+coords[i+1]
				p2 := []float64{cx, cy}
				path.AddStep(p1, p2)
			}
			cp = nil
		case 'C': // CubicTo
			for i := 0; i < len(coords); i += 6 {
				p1 := []float64{coords[i], coords[i+1]}
				p2 := []float64{coords[i+2], coords[i+3]}
				cp = p2
				p3 := []float64{coords[i+4], coords[i+5]}
				cx, cy = p3[0], p3[1]
				path.AddStep(p1, p2, p3)
			}
			qp = nil
		case 'c':
			for i := 0; i < len(coords); i += 6 {
				cxp, cyp := cx+coords[i], cy+coords[i+1]
				p1 := []float64{cxp, cyp}
				cxp, cyp = cx+coords[i+2], cy+coords[i+3]
				p2 := []float64{cxp, cyp}
				cp = p2
				cx, cy = cx+coords[i+4], cy+coords[i+5]
				p3 := []float64{cx, cy}
				path.AddStep(p1, p2, p3)
			}
			qp = nil
		case 'S': // SmoothCubicTo
			for i := 0; i < len(coords); i += 4 {
				// Infer p1 from reflected penultimate value of previous C/S step, else use current
				var p1 []float64
				if cp == nil {
					p1 = []float64{cx, cy}
				} else {
					dx, dy := cx-cp[0], cy-cp[1]
					p1 = []float64{cx + dx, cy + dy}
				}
				p2 := []float64{coords[i], coords[i+1]}
				cp = p2
				p3 := []float64{coords[i+2], coords[i+3]}
				cx, cy = p3[0], p3[1]
				path.AddStep(p1, p2, p3)
			}
			qp = nil
		case 's':
			for i := 0; i < len(coords); i += 4 {
				// Infer p1 from reflected penultimate value of previous C/S step, else use current
				var p1 []float64
				if cp == nil {
					p1 = []float64{cx, cy}
				} else {
					dx, dy := cx-cp[0], cy-cp[1]
					p1 = []float64{cx + dx, cy + dy}
				}
				cxp, cyp := cx+coords[i], cy+coords[i+1]
				p2 := []float64{cxp, cyp}
				cp = p2
				cx, cy = cx+coords[i+2], cy+coords[i+3]
				p3 := []float64{cx, cy}
				path.AddStep(p1, p2, p3)
			}
			qp = nil
		case 'A': // ArcTo
			for i := 0; i < len(coords); i += 7 {
				p1 := []float64{cx, cy}
				cx, cy = coords[i+5], coords[i+6]
				p2 := []float64{cx, cy}
				rx, ry := coords[i], coords[i+1]
				xang := coords[i+2] / 180 * math.Pi // value is in degrees
				la, swp := !util.Equals(coords[i+3], 0), !util.Equals(coords[i+4], 0)
				eap := g2d.EllipticalArcFromPoints2(p1, p2, rx, ry, xang, la, swp, g2d.ArcOpen)
				path.Concatenate(eap)
			}
			qp, cp = nil, nil
		case 'a':
			for i := 0; i < len(coords); i += 7 {
				p1 := []float64{cx, cy}
				cx, cy = cx+coords[i+5], cy+coords[i+6]
				p2 := []float64{cx, cy}
				rx, ry := coords[i], coords[i+1]
				xang := coords[i+2] / 180 * math.Pi // value is in degrees
				la, swp := !util.Equals(coords[i+3], 0), !util.Equals(coords[i+4], 0)
				eap := g2d.EllipticalArcFromPoints2(p1, p2, rx, ry, xang, la, swp, g2d.ArcOpen)
				path.Concatenate(eap)
			}
			qp, cp = nil, nil
		case 'Z':
			fallthrough
		case 'z':
			path.Close()
			res = append(res, path)
			cx, cy = cx0, cy0
			path = g2d.NewPath([]float64{cx, cy})
			newpath = true
			qp, cp = nil, nil
		default:
			fmt.Printf("%s not implemented yet\n", string(c))
		}
		first = false
	}
	if path != nil {
		res = append(res, path)
	}
	return res
}

// Parse into commands that start with one of ACHLMQSTVZ (and l/c versions)
func commands(str string) []string {
	// Convert all SVG ws types to space
	str = wscpat.ReplaceAllString(str, " ")

	// Chunk into path commands
	cmds := []string{"A", "a", "C", "c", "H", "h", "L", "l", "M", "m", "Q", "q", "S", "s", "T", "t", "V", "v", "Z", "z"}
	res := []string{str}
	for _, cmd := range cmds {
		tmp := []string{}
		for _, s := range res {
			tmp = append(tmp, splitByCmd(s, cmd)...)
		}
		res = tmp
	}

	// Normalize coordinates so space separated (because - is allowed)
	for i, s := range res {
		s = cpat.ReplaceAllString(s, "$1 -")
		s = adjpat.ReplaceAllString(s, "$1 $2")
		s = adjpat.ReplaceAllString(s, "$1 $2")
		res[i] = s
	}
	return res
}

func splitByCmd(str, cmd string) []string {
	strs := strings.Split(str, cmd)
	if len(strs) == 1 {
		// cmd not found
		return strs
	}
	res := make([]string, 0, len(strs))
	for i, s := range strs {
		if i == 0 && len(s) == 0 {
			continue
		}
		if i == 0 {
			res = append(res, s)
		} else {
			res = append(res, cmd+s)
		}
	}
	return res
}

func commandCoords(cmd string) (byte, []float64) {
	rem := cmd[1:]
	c := cmd[0]
	strs := strings.Split(rem, " ")
	if c == 'A' || c == 'a' {
		// Arc command needs to validate/handle flags
		ind := 0
		nstrs := make([]string, 0, (len(strs)/7+1)*7)
		for _, str := range strs {
			switch ind {
			case 3: // long arc flag
				l := len(str)
				switch l {
				case 1: // Well formed
					nstrs = append(nstrs, str)
					ind++
				case 2: // Mashed with sweep flag
					nstrs = append(nstrs, string(str[0]))
					nstrs = append(nstrs, string(str[1]))
					ind += 2
				default: // Mangled with sweep flag and x
					nstrs = append(nstrs, string(str[0]))
					nstrs = append(nstrs, string(str[1]))
					nstrs = append(nstrs, str[2:])
					ind += 3
				}
			case 4: // sweep flag
				l := len(str)
				switch l {
				case 1: // Well formed
					nstrs = append(nstrs, str)
					ind++
				default: // Mashed with x
					nstrs = append(nstrs, string(str[0]))
					nstrs = append(nstrs, str[1:])
					ind += 2
				}
			default:
				nstrs = append(nstrs, str)
				ind++
				if ind == 7 {
					ind = 0
				}
			}
		}
		strs = nstrs
	}
	res := make([]float64, 0, len(strs))
	for _, s := range strs {
		if len(s) == 0 {
			continue
		}
		var v float64
		n, err := fmt.Sscanf(s, "%f", &v)
		if n != 1 || err != nil {
			panic(err)
		}
		res = append(res, v)
	}
	return c, res
}

func ParseValue(str string) float64 {
	if str == "" {
		return 0
	}
	var v float64
	n, err := fmt.Sscanf(str, "%f", &v)
	if n != 1 || err != nil {
		panic("parse error")
	}
	return v
}

func ParseValueUnit(str string) (float64, string) {
	if str == "" {
		return 0, ""
	}
	var v float64
	var u string

	// TODO - this is hacky
	str = lcapat.ReplaceAllString(str, " $1")
	strs := strings.Split(str, " ")
	n, err := fmt.Sscanf(strs[0], "%f", &v)
	if n != 1 || err != nil {
		panic("parse error")
	}
	if len(strs) > 1 {
		u = strs[1]
	}
	return v, u
}

func ParseColor(str string) stdcol.Color {
	// empty or none
	if str == "" || str == "none" {
		return nil
	}
	// #XXX or #XXXXXX
	if strings.Index(str, "#") == 0 {
		str = str[1:]
		l := len(str)
		v := 0
		fmt.Sscanf(str, "%x", &v)
		switch l {
		case 3:
			r := ((v & 0xf00) >> 8) * 0x11
			g := ((v & 0xf0) >> 4) * 0x11
			b := (v & 0xf) * 0x11
			return stdcol.RGBA{uint8(r), uint8(g), uint8(b), 0xff}
		case 6:
			r := (v & 0xff0000) >> 16
			g := (v & 0xff00) >> 8
			b := v & 0xff
			return stdcol.RGBA{uint8(r), uint8(g), uint8(b), 0xff}
		default:
			return nil
		}
	}
	if strings.Index(str, "url") == 0 {
		// References gradients and patterns
		return nil
	}
	// rgb(...)
	strs := strings.Split(str, "(")
	if len(strs) == 2 {
		strs[1] = wscpat.ReplaceAllString(strs[1], " ")
		_, vals := commandCoords("X" + strs[1])
		return stdcol.RGBA{uint8(vals[0]), uint8(vals[1]), uint8(vals[2]), 0xff}
	}
	// Named color
	col, err := color.ByCSSName(str)
	if err != nil {
		return nil
	}
	return col.Color
}

func ParseStyle(str string, attrs map[string]string) {
	if str == "" {
		return
	}
	strs := strings.Split(str, ";")
	for _, str = range strs {
		if str == "" {
			continue
		}
		substrs := strings.Split(str, ":")
		// Assume 2 substrings
		attrs[substrs[0]] = substrs[1]
	}
}

func ParseViewBox(str string) [][]float64 {
	strs := strings.Split(str, " ")
	if len(strs) != 4 {
		return nil
	}
	x := ParseValue(strs[0])
	y := ParseValue(strs[1])
	dx := ParseValue(strs[2])
	dy := ParseValue(strs[3])
	return [][]float64{{x, y}, {x + dx, y + dy}}
}

func ParseTransform(str string) *g2d.Aff3 {
	if str == "" {
		return nil
	}

	cstrs := strings.Split(str, ")")
	commands := []string{}
	params := [][]float64{}
	for _, cstr := range cstrs {
		if cstr == "" {
			continue
		}
		parts := strings.Split(cstr, "(")
		if len(parts) == 0 || len(parts[0]) == 0 {
			continue
		}
		commands = append(commands, strings.TrimSpace(parts[0]))
		parts[1] = wscpat.ReplaceAllString(parts[1], " ")
		_, vals := commandCoords("X" + parts[1])
		params = append(params, vals)
	}

	// Construct transform
	xfm := g2d.NewAff3()

	for i, cmd := range commands {
		lp := params[i]
		switch cmd {
		case "matrix":
			xfm.Concatenate(g2d.Aff3{lp[0], lp[2], lp[4], lp[1], lp[3], lp[5]})
		case "translate":
			if len(lp) == 1 {
				xfm.Concatenate(*g2d.Translate(lp[0], 0))
			} else {
				xfm.Concatenate(*g2d.Translate(lp[0], lp[1]))
			}
		case "scale":
			if len(lp) == 1 {
				xfm.Concatenate(*g2d.Scale(lp[0], lp[0]))
			} else {
				xfm.Concatenate(*g2d.Scale(lp[0], lp[1]))
			}
		case "rotate":
			r := lp[0] * math.Pi / 180
			if len(lp) == 1 {
				xfm.Concatenate(*g2d.Rotate(r))
			} else {
				xfm.Concatenate(*g2d.RotateAbout(r, lp[1], lp[2]))
			}
		case "skewX":
			r := lp[0] * math.Pi / 180
			xfm.Concatenate(*g2d.Shear(math.Tan(r), 0))
		case "skewY":
			r := lp[0] * math.Pi / 180
			xfm.Concatenate(*g2d.Shear(0, math.Tan(r)))
		}
	}

	return xfm
}

func ParseUrlId(str string) string {
	// "url(#<...>)
	parts := strings.Split(str, "#")
	if len(parts) != 2 {
		return ""
	}
	lp2 := len(parts[1])
	return parts[1][0 : lp2-1]
}
