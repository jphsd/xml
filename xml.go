package xml

import (
	"encoding/xml"
	"io"
)

// XMLDecoder is a wrapper around xml.Decoder and holds the functions to be called when tokens are encountered.
// Functions that are left as nil are skipped by Process().
type XMLDecoder struct {
	Decoder      *xml.Decoder
	StartElement func(token xml.StartElement) error
	EndElement   func(token xml.EndElement) error
	CharData     func(token xml.CharData) error
	Comment      func(token xml.Comment) error
	ProcInst     func(token xml.ProcInst) error
	Directive    func(token xml.Directive) error
}

// NewXMLDecoder creates a new XMLDecoder that will read from the supplied io.Reader.
func NewXMLDecoder(r io.Reader) *XMLDecoder {
	return &XMLDecoder{xml.NewDecoder(r), nil, nil, nil, nil, nil, nil}
}

// Process performs the tokenization of the reader data and calls the user supplied functions.
func (d *XMLDecoder) Process() error {
	for {
		tok, err := d.Decoder.Token()
		if tok == nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch tok.(type) {
		case xml.StartElement:
			if d.StartElement != nil {
				se, _ := tok.(xml.StartElement)
				err = d.StartElement(se.Copy())
			}
		case xml.EndElement:
			if d.EndElement != nil {
				ee, _ := tok.(xml.EndElement)
				err = d.EndElement(ee)
			}
		case xml.CharData:
			if d.CharData != nil {
				cd, _ := tok.(xml.CharData)
				err = d.CharData(cd.Copy())
			}
		case xml.Comment:
			if d.Comment != nil {
				comm, _ := tok.(xml.Comment)
				err = d.Comment(comm.Copy())
			}
		case xml.ProcInst:
			if d.ProcInst != nil {
				pi, _ := tok.(xml.ProcInst)
				err = d.ProcInst(pi.Copy())
			}
		case xml.Directive:
			if d.Directive != nil {
				dir, _ := tok.(xml.Directive)
				err = d.Directive(dir.Copy())
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// BuildDOM inserts its own functions into the decoder in order to build the Domain Object Model.
func (d *XMLDecoder) BuildDOM() (*Element, error) {
	var root, cur *Element

	// Save existing functions
	sef := d.StartElement
	eef := d.EndElement
	cdf := d.CharData

	// Setup StartElement/EndElement/CharData
	d.StartElement = func(se xml.StartElement) error {
		if root == nil {
			root = &Element{Node, se.Name, make(map[string]string), nil, nil, nil}
			cur = root
		} else {
			tmp := &Element{Node, se.Name, make(map[string]string), nil, cur, nil}
			cur.Children = append(cur.Children, tmp)
			cur = tmp
		}
		for _, attr := range se.Attr {
			cur.Attributes[attr.Name.Local] = attr.Value
		}
		return nil
	}
	d.EndElement = func(ee xml.EndElement) error {
		cur = cur.Parent
		return nil
	}
	d.CharData = func(cd xml.CharData) error {
		if cur == nil {
			// Ignore CDATA outside of a Node
			return nil
		}
		tmp := &Element{Content, xml.Name{}, nil, cd, cur, nil}
		cur.Children = append(cur.Children, tmp)
		return nil
	}

	// Parse tokens into DOM tree
	err := d.Process()

	// Restore previous functions
	d.StartElement = sef
	d.EndElement = eef
	d.CharData = cdf

	if err != nil {
		return nil, err
	}
	return root, nil
}
