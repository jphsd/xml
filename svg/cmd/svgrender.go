//go:build ignore

package main

import (
	"bufio"
	"flag"
	//"fmt"
	g2d "github.com/jphsd/graphics2d"
	"github.com/jphsd/graphics2d/color"
	"github.com/jphsd/graphics2d/image"
	"github.com/jphsd/graphics2d/util"
	"github.com/jphsd/xml"
	"github.com/jphsd/xml/svg"
	"os"
)

// Read in a SVG file and render it to an image
func main() {
	imgf := flag.Bool("i", false, "use Image or Draw")
	clipf := flag.Bool("c", false, "overlay clip paths (-i only)")
	flag.Parse()

	// Get the file name from the command line or read stdin
	args := flag.Args()
	fn := "/dev/stdin"
	if len(args) > 0 {
		fn = args[0]
	}

	// Open file
	f, err := os.Open(fn)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Convert it to a domain object model
	decoder := xml.NewXMLDecoder(bufio.NewReader(f))
	dom, err := decoder.BuildDOM()
	if err != nil {
		panic(err)
	}

	if *imgf {
		// Turn svg into an image on a transparent background
		img, svgd := svg.Image(dom)

		if *clipf {
			cshape := &g2d.Shape{}
			for _, v := range svgd.Clip {
				cshape.AddShapes(v)
			}
			bounds := util.RectToBB(svgd.Rend.Bounds())
			xfm := g2d.Translate(-bounds[0][0], -bounds[0][1])
			cshape = cshape.Transform(xfm)
			g2d.DrawShape(img, cshape, g2d.BlackPen)
		}

		image.SaveImage(img, "svgrender-i")
	} else {
		// Create an image to render into to demonstrate scaling
		width, height := 1000, 1000
		img := image.NewRGBA(width, height, color.White)
		svg.Draw(img, dom)
		image.SaveImage(img, "svgrender-d")
	}
}
