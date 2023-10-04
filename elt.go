package xml

import (
	"encoding/xml"
)

// TT represents the element type.
type TT int

const (
	Node TT = iota
	Content
)

// Element is used to form the tree structure of the Document Object Model.
type Element struct {
	Type       TT                // Node or Content
	Name       xml.Name          // Node name
	Attributes map[string]string // Node attributes
	Content    xml.CharData      // CDATA content
	Parent     *Element          // Parent node
	Children   []*Element        // List of child nodes and contents for this node
}

// Copy returns a deep copy of this element and its children.
func (elt *Element) Copy() *Element {
	attrs := make(map[string]string)
	for k, v := range elt.Attributes {
		attrs[k] = v
	}

	nc := len(elt.Children)
	var children []*Element
	if nc > 0 {
		children = make([]*Element, nc)
		for i := 0; i < nc; i++ {
			children[i] = elt.Children[i].Copy()
		}
	}

	if elt.Content != nil {
		return &Element{elt.Type, elt.Name, attrs, elt.Content.Copy(), elt.Parent, children}
	}
	return &Element{elt.Type, elt.Name, attrs, nil, elt.Parent, children}
}
