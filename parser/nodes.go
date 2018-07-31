package runxml

import (
	"fmt"
)

// NodeType enum
type NodeType int

//go:generate stringer -type=NodeType
// NodeType enum values
const (
	Document    NodeType = iota //!< A document node. Name and value are empty.
	Element                     //!< An element node. Name contains element name. Value contains text of first data node.
	Data                        //!< A data node. Name is empty. Value contains data text.
	Cdata                       //!< A CDATA node. Name is empty. Value contains data text.
	Comment                     //!< A comment node. Name is empty. Value contains comment text.
	Declaration                 //!< A declaration node. Name and value are empty. Declaration parameters (version, encoding and standalone) are in node attributes.
	Doctype                     //!< A DOCTYPE node. Name is empty. Value contains DOCTYPE text.
	Pi                          //!< A PI node. Name contains target. Value contains instructions.
)

// Base is the common fields of nodes
type Base struct {
	Name   []byte       // Name of node
	Value  []byte       // Value of node
	Parent *GenericNode // Pointer to parent node
}

// GenericNode is the datastruct for all "node types" as defined above
type GenericNode struct {
	Base
	NodeType       NodeType       // NodeType enum (doc, element etc.)
	firstChild     *GenericNode   // pointer to first child node
	lastChild      *GenericNode   // pointer to last child node
	firstAttribute *AttributeNode // pointer to first attribute node
	lastAttribute  *AttributeNode // pointer to last attribute node
	prevSibling    *GenericNode   // pointer to previous sibling of node
	nextSibling    *GenericNode   // pointer to next sibling of node
}

// Mempool for allocation
var na NodeArena

func newNode(nodeType NodeType) *GenericNode {
	n := na.Get()
	n.NodeType = nodeType
	return n
}

// AppendAttribute appends an attribute to a node
func (g *GenericNode) AppendAttribute(a *AttributeNode) {
	if g.lastAttribute == nil {
		g.firstAttribute = a
		g.lastAttribute = a
	} else {
		a.prev = g.lastAttribute // point new attribute to the current last attribute
		g.lastAttribute.next = a // old a.next point to next attribute
		g.lastAttribute = a      // move pointer to last
	}
	a.Parent = g
}

// AppendNode appends a new child node to a node
func (g *GenericNode) AppendNode(child *GenericNode) {
	if g.firstChild != nil {
		// append
		g.lastChild.nextSibling = child
		child.prevSibling = g.lastChild
	} else {
		// its the first node
		g.firstChild = child
	}
	g.lastChild = child // lastnode is the last inserted
	child.Parent = g
}

// String representation of a GenericNode
func (g *GenericNode) String() string {
	return fmt.Sprintf("NodeType: \"%v\" Name: \"%s\" Value: \"%s\"\nParent: %p FirstNode: %p LastNode: %p FirstAttribute: %p LastAttribute: %p PrevSibling: %p NextSibling: %p",
		g.NodeType, g.Name, g.Value, g.Parent, g.firstChild, g.lastChild, g.firstAttribute, g.lastAttribute, g.prevSibling, g.nextSibling)
}

// PrintChildren prints a representation of the node, including its children
func (g *GenericNode) PrintChildren() {
	var depthmap map[*GenericNode]int
	depthmap = make(map[*GenericNode]int)

	for n := range g.SendChildElements() {
		if n.Parent == nil {
			depthmap[n] = 0
		}
		var depth int
		var ok bool
		if depth, ok = depthmap[n]; !ok {
			depthmap[n] = depthmap[n.Parent] + 1
			depth = depthmap[n]
		}
		indent := ""
		for i := 0; i < depth; i++ {
			indent += "-"
		}
		fmt.Print(indent)
		fmt.Printf("NodeType: \"%v\" Name: \"%s\" Value: \"%s\" Parent: %p FirstNode: %p\n", n.NodeType, n.Name, n.Value, n.Parent, n.firstChild)
		// Print Attributes
		fmt.Print(indent)
		for _, a := range n.GetAttributes() {
			fmt.Println(a)
		}
	}
}

//CountChildren returns the current nodes number of child nodes
func (g *GenericNode) CountChildren() int {
	count := 0
	for range g.SendChildElements() {
		count++
	}
	return count
}

// SendCloseChildren returns a channel of pointers to
// all direct children, but not their children.
func (g *GenericNode) SendCloseChildren() (ret chan *GenericNode) {
	ret = make(chan *GenericNode, 100)
	// traverse the siblings of the child
	go func() {
		for n := g.firstChild; ; n = n.nextSibling {
			ret <- n
			// if it is the last node, break
			if n == g.lastChild {
				break
			}
		}
		close(ret)
	}()
	return ret
}

// SendChildElements returns a channel of pointers to
// itself and all child elements of the node
func (g *GenericNode) SendChildElements() (ret chan *GenericNode) {
	if g == nil {
		panic("node is nil")
	}
	ret = make(chan *GenericNode, 100)
	var trav func(g *GenericNode)
	trav = func(g *GenericNode) {
		ret <- g // return yoursself
		if g.firstChild != nil {
			trav(g.firstChild)
		}
		// no more children, look for next sibling
		if g.nextSibling != nil {
			trav(g.nextSibling)
		}
		// no more siblings
		return
	}
	go func() {
		trav(g)
		close(ret)
	}()
	return
}

// GetFirstChild returns the first child of the node,
// if no child exists, it returns null
func (g *GenericNode) GetFirstChild() *GenericNode {
	return g.firstChild
}

// GetLastChild returns the last child of the node,
// if no child exists, it returns null
func (g *GenericNode) GetLastChild() *GenericNode {
	return g.lastChild
}

// GetNextSibling returns the next sibling of the node,
// if no siblings exist, or we reached the end of the siblings
// it returns null
func (g *GenericNode) GetNextSibling() *GenericNode {
	return g.nextSibling
}

// GetPreviousSibling returns the previous sibling of the node,
// if no siblings exist, or we reached the first of the siblings
// it returns null
func (g *GenericNode) GetPreviousSibling() *GenericNode {
	return g.prevSibling
}

// GetAttributes returns a slice of pointers to the attributes of the node
func (g *GenericNode) GetAttributes() []*AttributeNode {
	retAttrrib := make([]*AttributeNode, 0, 10)
	for i := g.firstAttribute; i != nil; i = i.next {
		retAttrrib = append(retAttrrib, i)
	}
	return retAttrrib
}

// AttributeNode represents the attribute (a="abc") of a node
type AttributeNode struct {
	Base
	prev *AttributeNode
	next *AttributeNode
}

// String representation of a attribute node
func (a *AttributeNode) String() string {
	return fmt.Sprintf("Attribute Name: \"%s\" Value: \"%s\" Parent: %p Prev: %p Next: %p",
		a.Name, a.Value, a.Parent, a.prev, a.next)
}
