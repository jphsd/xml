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
	// Get the file name from the command line or read stdin
	flag.Parse()
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

	// Create an image to render into
	width, height := 1000, 1000
	img := image.NewRGBA(width, height, color.White)

	// Render it using the DOM
	renderer := svg.NewSVG(img)
	renderer.Process(dom)

	// Save the result
	image.SaveImage(img, "svgrender-1")

	// Save the renderables (viewBox independent)
	img = image.NewRGBA(width, height, color.White)
	renderer.Rend.Render(img, nil)
	image.SaveImage(img, "svgrender-2")
}
