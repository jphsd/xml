# xml
[![Go Reference](https://pkg.go.dev/badge/github.com/jphsd/xml.svg)](https://pkg.go.dev/github.com/jphsd/xml)
[![godocs.io](http://godocs.io/github.com/jphsd/xml?status.svg)](http://godocs.io/github.com/jphsd/xml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jphsd/xml)](https://goreportcard.com/report/github.com/jphsd/xml)

Wrapper around encoding/xml.Decode to facilitate the Inversion of Control pattern and provide a domain object model builder.

The enclosed svg package is a simplistic SVG11 renderer that can parse a DOM from the above. The following elements are (mostly) supported:
  svg
  defs
  use
  g
  line
  rect
  circle
  ellipse
  polyline
  polygon
  path
  clipPath

The svgrender command (xml/svg/cmd) can read either the standard input or the supplied file and render it to svgrender.png.
