//go:build ignore

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/jphsd/xml"
	"os"
	"strings"
)

// Read in an XML file
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

	dump(dom, 0)
}

func dump(dom *xml.Element, indent int) {
	switch dom.Type {
	case xml.Node:
		res := makeInd(indent) + dom.Name.Local + ": "
		for k, v := range dom.Attributes {
			res += k + "=" + v + " "
		}
		fmt.Println(res)
		for _, c := range dom.Children {
			dump(c, indent+1)
		}
	case xml.Content:
		txt := strings.Trim(string(dom.Content), " \t\n")
		if len(txt) != 0 {
			res := makeInd(indent) + txt
			fmt.Println(res)
		}
	}
}

func makeInd(i int) string {
	ind := "  "
	res := ""
	for ; i > 0; i-- {
		res += ind
	}
	return res
}
