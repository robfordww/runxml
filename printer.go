package runxml

import "fmt"

// Document    NodeType = iota //!< A document node. Name and value are empty.
// 	Element                     //!< An element node. Name contains element name. Value contains text of first data node.
// 	Data                        //!< A data node. Name is empty. Value contains data text.
// 	Cdata                       //!< A CDATA node. Name is empty. Value contains data text.
// 	Comment                     //!< A comment node. Name is empty. Value contains comment text.
// 	Declaration                 //!< A declaration node. Name and value are empty. Declaration parameters (version, encoding and standalone) are in node attributes.
// 	Doctype                     //!< A DOCTYPE node. Name is empty. Value contains DOCTYPE text.
// 	Pi                          //!< A PI node. Name contains target. Value contains instructions.

// PrintXML writes to stdout an XML representation of the node structure.
func (g *GenericNode) PrintXML() {
	p := printer{pretty: false}
	p.printStructure(g)
}

// PrintXMLPretty writes to stdout an XML representation of the node structure and inserting
// indenting and line breaking characters for prettier formatting
func (g *GenericNode) PrintXMLPretty() {
	p := printer{pretty: true}
	p.printStructure(g) // Not implemented yet
}

// printer holds variables for printer settings
type printer struct {
	pretty      bool
	indentvalue int
}

// PrintXML writes a textual representation from children of g
func (p *printer) printStructure(gn *GenericNode) {
	// traverse siblings
	for s := gn; s != nil; s = s.next {
		switch s.NodeType {
		case Declaration:
			// print attributes
		case Element:
			if p.pretty {
				fmt.Println("")
			}
			// can have children and siblings which must be handled
			fmt.Print("<" + string(s.Name) + ">")
			p.traverseDepth(s)
			fmt.Print("</" + string(s.Name) + ">")
		case Data:
			// just print and return
			fmt.Print(string(s.Value))
		case Cdata:
			//  cdata needs to be embedded in a CDATA structure
			fmt.Print(`<![CDATA[` + string(s.Value) + `]]`)
		case Comment:
			fmt.Print("<!--" + string(s.Value) + "-->")
		case Doctype:
			fmt.Print("<!DOCTYPE " + string(s.Value) + ">")
			p.traverseDepth(s)
		case Pi:
			fmt.Print("<?" + string(s.Name) + " " + string(s.Value))
		case Document:
			p.traverseDepth(s)
		default:
			panic("unknown node type")
		}

	}

}

func (p *printer) traverseDepth(g *GenericNode) {
	if g.firstChild != nil {
		p.indentvalue++
		p.printStructure(g.firstChild)
	}
}
