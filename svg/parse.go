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
	wspat  = regexp.MustCompile(`[ \n\r\t]+`)  // Whitespace pattern
	wscpat = regexp.MustCompile(`[, \n\r\t]+`) // Whitespace and comma pattern
	cpat   = regexp.MustCompile(`(\d)-`)       // Digit follwed by -ve sign pattern (SVG allows + too...)
	lcapat = regexp.MustCompile(`([a-z]+)`)    // Lower case word pattern
)

func PathsFromDescription(desc string) []*g2d.Path {
	cmds := commands(desc)
	cx, cy := 0.0, 0.0
	res := []*g2d.Path{}
	var path *g2d.Path
	var cp, qp []float64
	for _, cmd := range cmds {
		c, coords := commandCoords(cmd)
		switch c {
		case 'M': // MoveTo
			if path != nil {
				res = append(res, path)
			}
			for i := 0; i < len(coords); i += 2 {
				cx, cy = coords[i], coords[i+1]
				if i == 0 {
					path = g2d.NewPath([]float64{cx, cy})
				} else {
					// Additional pairs treated as L
					path.AddStep([]float64{cx, cy})
				}
			}
			qp, cp = nil, nil
		case 'm':
			if path != nil {
				res = append(res, path)
			}
			for i := 0; i < len(coords); i += 2 {
				cx, cy = cx+coords[i], cy+coords[i+1]
				if i == 0 {
					path = g2d.NewPath([]float64{cx, cy})
				} else {
					// Additional pairs treated as l
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
			path = nil
			qp, cp = nil, nil
		default:
			fmt.Printf("%s not implemented yet\n", string(c))
		}
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
	tmp := res
	res = make([]string, len(tmp))
	for i, s := range tmp {
		res[i] = cpat.ReplaceAllString(s, "$1 -")
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
	// none
	if str == "none" {
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
	// Named color
	col, err := color.ByName(str)
	if err != nil {
		return nil
	}
	return col.Color
}

func ParseStyle(str string, attrs map[string]string) {
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
