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
func (r *GenericNode) PrintXML() {
	for c := range r.SendChildElements() {
		switch c.NodeType {
		case Document:
			continue // Nothing to print
		case Declaration:
			continue // print attributes
		case Element:
			fmt.Print("<" + string(c.Name) + ">" + string(c.Value) + "</" + string(c.Name) + ">")
		case Data:
			fmt.Print(c.Value)
		case Cdata:
			fmt.Print(c.Value)
		case Comment:
			fmt.Print("<!--" + string(c.Value) + "-->")
		case Doctype:
			fmt.Print("<!DOCTYPE " + string(c.Value) + ">")
		case Pi:
			fmt.Print("<?" + string(c.Name) + " " + string(c.Value))
		}
	}
}
