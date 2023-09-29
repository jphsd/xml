//go:build ignore

package main

import (
	"bufio"
	"flag"
	//"fmt"
	"github.com/jphsd/graphics2d/color"
	"github.com/jphsd/graphics2d/image"
	"github.com/jphsd/xml"
	"github.com/jphsd/xml/svg"
	"os"
)

// Read in a SVG file and render it to an image
func main() {
	flag.Parse()
	args := flag.Args()
	fn := "/dev/stdin"
	if len(args) > 0 {
		fn = args[0]
	}

	f, err := os.Open(fn)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	decoder := xml.NewXMLDecoder(bufio.NewReader(f))

	dom, err := decoder.BuildDOM()
	if err != nil {
		panic(err)
	}

	width, height := 1000, 1000
	img := image.NewRGBA(width, height, color.White)

	renderer := svg.NewSVG(img)
	renderer.Process(dom)

	image.SaveImage(img, "svgrender")
}
