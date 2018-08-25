// Package runxml is a fast xml parser based on the RapidXML C++ library.
//
// Runxmls goal is to a provide a fast DOM parser and marshaller/unmarhaller
// by using in-situ parsing and static generated code rather than reflection.

//go:generate stringer -type=NodeType

package runxml

import (
	"fmt"
)

// NodeType is the datatype descriping all possible node types
type NodeType int

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

// base contains the common fields of nodes
type base struct {
	Name   []byte       // Name of node
	Value  []byte       // Value of node
	Parent *GenericNode // Pointer to parent node
}

// GenericNode is the datastruct for all "node types" as defined above
type GenericNode struct {
	base
	NodeType       NodeType       // NodeType enum (doc, element etc.)
	firstChild     *GenericNode   // pointer to first child node
	lastChild      *GenericNode   // pointer to last child node
	firstAttribute *AttributeNode // pointer to first attribute node
	lastAttribute  *AttributeNode // pointer to last attribute node
	prev           *GenericNode   // pointer to previous sibling of node
	next           *GenericNode   // pointer to next sibling of node
}

// Mempool for allocation
var na nodeArena

func newNode(nodeType NodeType) *GenericNode {
	n := na.get()
	n.NodeType = nodeType
	return n
}

// AppendNode appends a new child node to a node
func (g *GenericNode) AppendNode(child *GenericNode) {
	if g.firstChild == nil {
		// it becomes the first child if no nodes
		// exist
		g.firstChild = child
	} else {
		// append to existing structure
		g.lastChild.next = child
		child.prev = g.lastChild
	}
	g.lastChild = child // lastnode is the last inserted
	child.Parent = g
}

// PrependNode inserts a child as the first node
func (g *GenericNode) PrependNode(child *GenericNode) {
	if g.firstChild == nil {
		// special case if there are no attributes
		// otherwise, last is not affected
		g.lastChild = child
	} else {
		// first update the existing child
		g.firstChild.prev = child
		child.next = g.firstChild
	}
	g.firstChild = child // lastnode is the last inserted
	child.Parent = g
}

// InsertNode inserts a child before the specified child node
func (g *GenericNode) InsertNode(where, child *GenericNode) {
	if g.firstChild == where {
		g.PrependNode(child)
		return
	}
	if where.Parent != g {
		panic("attempted to insert at non-existing position")
	}
	child.prev = where.prev
	child.next = where
	where.prev.next = child
	where.prev = child
	child.Parent = g
}

// RemoveFirstNode deletes the first child of the node
func (g *GenericNode) RemoveFirstNode() {
	if g.firstChild == nil {
		return // nothing to remove
	}
	g.firstChild = g.firstChild.next
	if g.firstChild == nil {
		// no children left, update lastchild to nil too
		g.lastChild = nil
		return
	}
	g.firstChild.prev = nil
}

// RemoveLastNode deletes the last child of the node
func (g *GenericNode) RemoveLastNode() {
	if g.lastChild == nil {
		return // nothing to remove
	}
	g.lastChild = g.lastChild.prev
	if g.lastChild == nil {
		// no children left, update first child too
		g.firstChild = nil
		return
	}
	g.lastChild.next = nil
}

// RemoveNode deletes a particular child node of the current node
func (g *GenericNode) RemoveNode(where *GenericNode) {
	if where == g.firstChild {
		g.RemoveFirstNode()
		return
	} else if where == g.lastChild {
		g.RemoveLastNode()
		return
	}
	if where.Parent != g {
		panic("attempting to remove non-child node")
	}
	// splice where's siblings
	where.prev.next = where.next
	where.next.prev = where.prev
}

// RemoveAllNodes removes all child nodes, but not attributes of the current node
func (g *GenericNode) RemoveAllNodes() {
	g.firstChild = nil
	g.lastChild = nil
}

// String representation of a GenericNode
func (g *GenericNode) String() string {
	return fmt.Sprintf("NodeType: \"%v\" Name: \"%s\" Value: \"%s\"\nParent: %p FirstNode: %p LastNode: %p FirstAttribute: %p LastAttribute: %p PrevSibling: %p NextSibling: %p",
		g.NodeType, g.Name, g.Value, g.Parent, g.firstChild, g.lastChild, g.firstAttribute, g.lastAttribute, g.prev, g.next)
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

// SendCloseChildren returns a channel of pointers to  all direct children,
// but not their children. This is useful for breadth first parsing
func (g *GenericNode) SendCloseChildren() (ret chan *GenericNode) {
	ret = make(chan *GenericNode, 8)
	// traverse the siblings of the child
	go func(ret chan<- *GenericNode) {
		if g.firstChild == nil {
			// no children
			close(ret)
			return
		}
		for n := g.firstChild; ; n = n.next {
			ret <- n
			// if it is the last node, break
			if n == g.lastChild {
				break
			}
		}
		close(ret)
	}(ret)
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
		if g.next != nil {
			trav(g.next)
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
	return g.next
}

// GetPreviousSibling returns the previous sibling of the node,
// if no siblings exist, or we reached the first of the siblings
// it returns null
func (g *GenericNode) GetPreviousSibling() *GenericNode {
	return g.prev
}

// GetAttributes returns a slice of pointers to the attributes of the node
func (g *GenericNode) GetAttributes() []*AttributeNode {
	retAttrrib := make([]*AttributeNode, 0, 10)
	for i := g.firstAttribute; i != nil; i = i.next {
		retAttrrib = append(retAttrrib, i)
	}
	return retAttrrib
}

// AppendAttribute appends an attribute to a node
func (g *GenericNode) AppendAttribute(a *AttributeNode) {
	if g.firstAttribute == nil {
		g.firstAttribute = a
	} else {
		g.lastAttribute.next = a // old a.next point to next attribute
		a.prev = g.lastAttribute // point new attribute to the current last attribute
	}
	g.lastAttribute = a // move pointer to last
	a.Parent = g
}

// PrependAttribute prepends an attribute to the current node
func (g *GenericNode) PrependAttribute(a *AttributeNode) {
	if g.firstAttribute == nil {
		// special case if there are no attributes
		// otherwise, last is not affected
		g.lastAttribute = a
	} else {
		// update the existing first's previous node
		g.firstAttribute.prev = a
		a.next = g.firstAttribute
	}
	g.firstAttribute = a
	g.Parent = g
}

// InsertAttribute inserts an attribute before the specified
// child node
func (g *GenericNode) InsertAttribute(where, a *AttributeNode) {
	if g.firstAttribute == where {
		g.PrependAttribute(a)
	} else if where.Parent != g {
		panic("attemted to insert attribute at non-exising position")
	} else {
		a.prev = where.prev
		a.next = where
		where.prev.next = a
		where.prev = a
		a.Parent = g
	}
}

// AttributeNode represents the attribute (a="abc") of a node
type AttributeNode struct {
	base
	prev *AttributeNode
	next *AttributeNode
}

// String representation of a attribute node
func (a *AttributeNode) String() string {
	return fmt.Sprintf("Attribute Name: \"%s\" Value: \"%s\" Parent: %p Prev: %p Next: %p",
		a.Name, a.Value, a.Parent, a.prev, a.next)
}

// --function prepend_node(xml_node< Ch > *child)
// --function append_node(xml_node< Ch > *child)
// --function insert_node(xml_node< Ch > *where, xml_node< Ch > *child)
// -- function remove_first_node()
// -- function remove_last_node()
// -- function remove_node(xml_node< Ch > *where)
// -- function remove_all_nodes()
// function prepend_attribute(xml_attribute< Ch > *attribute)
// function append_attribute(xml_attribute< Ch > *attribute)
// function insert_attribute(xml_attribute< Ch > *where, xml_attribute< Ch > *attribute)
// function remove_first_attribute()
// function remove_last_attribute()
// function remove_attribute(xml_attribute< Ch > *where)
// function remove_all_attributes()
