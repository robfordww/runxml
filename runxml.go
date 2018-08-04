package runxml

import (
	"bytes"
	"fmt"
	"io/ioutil"
)

// RunXML is the parser instance that tracks the holds all state info
type RunXML struct {
	ValidateClosingTag bool
	nodeArena          nodeArena      // Optimizing memory allocations
	attributeArena     attributeArena // Optimizing memory allocations
	data               []byte         // Data buffer
	position           int            // Internal read position
	// Config settings
}

// NewDefaultRunXML creates a standard parser setup.
func NewDefaultRunXML() *RunXML {
	r := new(RunXML)
	r.ValidateClosingTag = true
	return r
}

// ParseFile is a wrapper for Parse to simplify loading of files
func (r *RunXML) ParseFile(fn string) (*GenericNode, error) {
	bs, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	return r.Parse(bs)
}

// Parse parses the entire byte slice.
// Returns a pointer to GenericNode, representing the entire XML DOM-tree
func (r *RunXML) Parse(b []byte) (*GenericNode, error) {
	r.position = 0
	r.data = b
	doc := newNode(Document)
	// Skip possible BOM
	r.skipBOM()
	for r.position < len(r.data) {
		// skip spaces
		r.skip(lookupWhitespace)
		if r.position == len(r.data)-1 {
			break // normal end of file
		}
		c := r.getCurrentByte()
		if c == '<' {
			r.position++
			node, err := r.parseNode()
			if err != nil {
				return node, r.contextError(err)
			}
			doc.AppendNode(node)
		} else {
			return doc, r.contextError(fmt.Errorf("expected '<', but found %q", rune(r.data[r.position])))
		}
	}
	return doc, nil
}

// parseNode is the highest level parsing method; expects position to be after a '<'
func (r *RunXML) parseNode() (*GenericNode, error) {
	//log.Println("parsing node at position", r.position, string(r.sliceForward(20)))
	c := r.data[r.position]
	switch c {
	// <?...
	case '?':
		if err := r.skipBytes(4); err != nil {
			return nil, fmt.Errorf("unexpected end of file")
		}
		x := r.sliceFrom(r.position - 3)
		//log.Println("PARSEX", string(x))
		if bytes.Compare([]byte("xml"), bytes.ToLower(x)) == 0 &&
			lookupWhitespace[r.getCurrentByte()] == 1 {
			r.getNextByte() // skip to next byte
			//log.Println("PARSE")
			return r.parseXMLDeclaration()
		}
		r.position -= 3 // go back 4
		// not <?xml, so parse program instruction
		return r.parsePI()
	case '!':
		//fmt.Println("encounter !")
		// Parse proper subset
		switch c2 := r.getNextByte(); c2 {
		// <![
		case '-':
			//fmt.Println("encounter -")
			if c2 := r.getNextByte(); c2 == '-' {
				r.getNextByte() // <!--
				//fmt.Println("parse comment")
				return r.parseComment()
			}

		// <![
		case '[':
			err := r.skipBytes(1)
			if err != nil {
				return nil, err
			}
			// <![CDATA[]
			if !bytes.HasPrefix(r.sliceToEnd(), []byte("CDATA[")) {
				return nil, fmt.Errorf("unexpecte data following <![")
			}
			r.skipBytes(6) // skip <![CDATA[
			return r.parseCDATA()
		// <!D
		case 'D':
			err := r.skipBytes(1)
			if err != nil {
				return nil, err
			}
			if bytes.HasPrefix(r.sliceToEnd(), []byte("OCTYPE")) && lookupWhitespace[r.data[r.position+6]] == 1 {
				// "<!DOCTYPE "
				r.skipBytes(6)
				return r.parseDocType()
			}
			fallthrough //? needed ?
		case 0: // zerobyte returned, not legal
			return nil, fmt.Errorf("unexpected end of file at %v", r.position)
		default: // Attempt to skip other, unrecognized node types starting with <!
			err := r.skipPastChar('>')
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("unrecognized node at %v", r.position)
		}
	default:
		// log.Println("Parselement")
		return r.parseElement()
	}

	/*// skip undefined node types <!
	if err := r.skipBytes(1); err != nil {
		return nil, err
	}*/
	err := r.skipToChar('>')
	return nil, err
}

// parseAttributes parses the attribute of an element, returns a AttributeNode
func (r *RunXML) parseAttributes(element *GenericNode) error {
	// repeat for each attribute
	for lookupAttributeName[r.getCurrentByte()] == 1 {
		start := r.position
		r.position++
		r.skip(lookupAttributeName)
		attrNode := r.attributeArena.get() // Fetch new node
		attrNode.Name = r.sliceFrom(start)
		element.AppendAttribute(attrNode)

		// skip whitespace
		r.skip(lookupWhitespace)
		// skip "="
		if r.getCurrentByte() != '=' {
			return fmt.Errorf("expected '=' but found %q at position %v", r.data[r.position], r.position)
		}
		r.position++

		// skip whitespace after =
		r.skip(lookupWhitespace)
		q := r.getCurrentByte()
		if q != '\'' && q != '"' {
			return fmt.Errorf("expected ' or \" but found %q at position %v", q, r.position)
		}
		r.position++ // Skip quote
		// Extract attribute value, and expand char refs in it
		start = r.position
		var value []byte
		if q == '\'' {
			value = r.skipAndExpandCharacterRefs(lookupAttributeDataSQ, lookupAttributeDataSQPure)
		} else if q == '"' {
			value = r.skipAndExpandCharacterRefs(lookupAttributeDataDQ, lookupAttributeDataDQPure)
		} else {
			panic("should never happen")
		}
		if value == nil {
			return fmt.Errorf("error parsing attribute value")
		}
		// Set attribute value
		attrNode.Value = value
		// Make sure end quote is present
		if r.getCurrentByte() != q {
			return fmt.Errorf("expected %v as end quote", q)
		}
		r.position++ // skip quote
		// Skip whitespace after attribute value
		r.skip(lookupWhitespace)
	}
	return nil
}

// parseElement parses element node
func (r *RunXML) parseElement() (*GenericNode, error) {
	//fmt.Println("parse elem", r.position)
	currentElement := newNode(Element)
	// Extract element name
	start := r.position
	r.skip(lookupNodeName)
	if start == r.position {
		return nil, fmt.Errorf("error parsing node name")
	}
	//log.Println("parse elem post lookup", r.position)
	currentElement.Name = r.data[start:r.position]
	//fmt.Println("DEBUG:", string(currentElement.Name))

	// Skip whitespace between element name and attributes or >
	r.skip(lookupWhitespace)

	// Parse attributes
	err := r.parseAttributes(currentElement)
	if err != nil {
		return nil, err
	}

	// determine ending type
	c := r.getCurrentByte()
	if c == '>' {
		r.position++
		if err := r.parseNodeContents(currentElement); err != nil {
			return nil, err
		}
	} else if c == '/' {
		if r.getNextByte() != '>' {
			return nil, fmt.Errorf("expected '>' after '/' at position %v", r.position)
		}
		r.position++
	} else {
		return nil, fmt.Errorf("unknown end type error")
	}
	return currentElement, nil
}

// parseNodeContents Parse contents of the node - children, data etc.
func (r *RunXML) parseNodeContents(cn *GenericNode) error {
	// For all children and text
	for {
		//contentStart := r.position
		r.skip(lookupWhitespace)
	AfterDataNode:
		c := r.getCurrentByte()
		// Determine what comes next: node closing, child node, data node, or 0?
		switch c {
		// New child node, closing, or data
		case '<': // closing, child or error
			if r.getNextByte() == '/' {
				// Node closing
				r.position++ // Skip to first char of closing tag
				if r.ValidateClosingTag {
					start := r.position
					r.skip(lookupNodeName)
					closeTag := r.sliceFrom(start)
					if bytes.Compare(closeTag, cn.Name) != 0 {
						return fmt.Errorf("unexpected closing tag %v", closeTag)
					}
				} else {
					r.skip(lookupNodeName) // close regardless
				}
				r.skip(lookupWhitespace) // Skip remaining whitespace after nodename
				if r.getCurrentByte() != '>' {
					return fmt.Errorf("expected '>'")
				}
				r.position++ // Skip '>'
				return nil
			}
			// Child node
			//log.Println("child node")
			child, err := r.parseNode()
			if err != nil {
				return err
			}
			if child != nil {
				cn.AppendNode(child)
			}

		// Data node in node, create data node
		default:
			err := r.parseAndAppendData(cn)
			if err != nil {
				return err
			}
			goto AfterDataNode
		}
	}
}

// parseDocType returns the Doctype Node
func (r *RunXML) parseDocType() (*GenericNode, error) {
	start := r.position
	// skip to > , we haven't closed the tag yet
	// since doctype can contain other elements, it can be somewhat tricky to detect in an efficient
	// way when it ends
	for r.getCurrentByte() != '>' {
		if r.getCurrentByte() == '[' { // beginning of elements
			r.skipBytes(1) // skip the '['
			//quoteType := '"'
			for depth, insideElement := 1, false; depth > 0; {
				switch r.getCurrentByte() {
				case '[':
					if !insideElement { // only count if not in a quote
						depth++
					}
				case ']':
					if !insideElement {
						depth--
					}
				case '>':
					insideElement = false
				}
				if bytes.HasPrefix(r.sliceToEnd(), []byte("<!")) {
					insideElement = true
				}
				r.getNextByte()
			}
		} else {
			err := r.skipBytes(1)
			if err != nil {
				return nil, err
			}
		}
	}
	dt := newNode(Doctype)
	dt.Value = r.sliceFrom(start)
	r.skipBytes(1)
	return dt, nil
}

func (r *RunXML) parseXMLDeclaration() (*GenericNode, error) {
	nd := newNode(Declaration)
	r.skip(lookupWhitespace)
	r.parseAttributes(nd)
	// expect closing tags after attributes
	if !bytes.HasPrefix(r.sliceToEnd(), []byte("?>")) {
		r.position += 2
		return nil, fmt.Errorf("unexpected end of xml declaration. Expected '?>'")
	}
	r.position += 2
	return nd, nil
}

// parsePI returns a Program Instruction node
func (r *RunXML) parsePI() (*GenericNode, error) {
	start := r.position
	r.skip(lookupNodeName)
	if start == r.position {
		return nil, fmt.Errorf("expected PI target")
	}
	pin := newNode(Pi)
	pin.Name = r.sliceFrom(start)
	r.skip(lookupWhitespace)
	start = r.position
	// skip to ?>
	if err := r.skipToChars([]byte("?>")); err != nil {
		return nil, err
	}
	pin.Value = r.sliceFrom(start)
	r.position += 2
	return pin, nil
}

// parseCDATA creates a CDATA node
func (r *RunXML) parseCDATA() (*GenericNode, error) {
	start := r.position // expects after <![CDATA[
	err := r.skipToChars([]byte("]]"))
	if err != nil {
		return nil, err
	}
	cd := newNode(Cdata)
	cd.Value = r.sliceFrom(start)
	return cd, nil
}

// parseComment creates the Comment node
func (r *RunXML) parseComment() (*GenericNode, error) {
	//log.Println("parse comment")
	start := r.position
	// Skip to end of comments
	for !bytes.HasPrefix(r.sliceToEnd(), []byte("--")) {
		if err := r.skipBytes(1); err != nil {
			return nil, fmt.Errorf("unexpected end of file")
		}
	}
	if err := r.skipBytes(2); err != nil {
		return nil, fmt.Errorf("unexpected end of file")
	}
	if r.getCurrentByte() != '>' {
		// there is '--' inside comment; not allowed in specs.
		return nil, fmt.Errorf("invalid '--' inside comment")
	}
	comment := newNode(Comment)
	comment.Value = r.data[start : r.position-2]
	//log.Printf("DEBUG: %#v\n", comment)
	r.skipBytes(1)
	return comment, nil
}

func (r *RunXML) getCurrentByte() byte {
	return r.data[r.position]
}

func (r *RunXML) getNextByte() byte {
	r.position++
	if r.position > len(r.data)-1 {
		return 0
	}
	return r.data[r.position]
}

func (r *RunXML) skipBytes(n int) error {
	if len(r.data) <= r.position+n {
		return fmt.Errorf("%v", "end of file")
	}
	r.position += n
	return nil
}

func (r *RunXML) skipPastChar(b byte) error {
	for {
		r.position++
		if r.position >= len(r.data)-1 { // if, then we are at last char and cant advance
			return fmt.Errorf("unexpected end of data")
		}
		if r.data[r.position] == b {
			r.position++ // advance, we know there is enough room
			return nil
		}
	}
}

func (r *RunXML) skipToChar(b byte) error {
	for {
		r.position++
		if r.position >= len(r.data) { // if, then we are at last char and cant advance
			return fmt.Errorf("unexpected end of data")
		}
		if r.data[r.position] == b {
			return nil
		}
	}
}

func (r *RunXML) skipToChars(b []byte) error {
	for ; r.position < len(r.data); r.position++ {
		if bytes.HasPrefix(r.data[r.position:], b) {
			return nil
		}
	}
	r.position-- // lower position to not crash at end of data
	return fmt.Errorf("unexpected end of file")
}

// skip characters until table evaluates to true, then return offset
func (r *RunXML) skip(table *[256]byte) {
	for ; r.position < len(r.data); r.position++ {
		if table[r.data[r.position]] != 1 {
			return
		}
	}
	r.position-- // lower position to not crash at end of data
}

func (r *RunXML) skipBOM() {
	// UTF8
	if bytes.HasPrefix(r.data, []byte{0xEF, 0xBB, 0xBF}) {
		r.position += 3
	} else if bytes.HasPrefix(r.data, []byte{0xFF, 0xFE}) {
		//log.Println("warning, utf16le")
		// UTF 16 LE
		var err error
		r.data, err = decodeUTF16(r.data)
		if err != nil {
			panic(err)
		}
		r.position += 3
	} else if bytes.HasPrefix(r.data, []byte{0xFE, 0xFF}) {
		//log.Println("warning, utf16be")
		// UTF 16 BE
		var err error
		r.data, err = decodeUTF16(r.data)
		if err != nil {
			panic(err)
		}
		r.position += 3
	}

}

func (r *RunXML) sliceFrom(start int) []byte {
	return r.data[start:r.position]
}

func (r *RunXML) sliceForward(i int) []byte {
	if r.position+i <= len(r.data) {
		return r.data[r.position : r.position+i]
	}
	return r.data[r.position:len(r.data)]
}

func (r *RunXML) sliceToEnd() []byte {
	return r.data[r.position:]
}

// skip and expand charaters is both used to parse attribute values and node data while expanding entities
// since this function can overwrite the buffer, it returns a slice of the active area
func (r *RunXML) skipAndExpandCharacterRefs(stopPred, stopPredPure *[256]byte) []byte {
	start := r.position
	r.skip(stopPredPure) // fast path if no '&' is found
	trail := r.position
	for c := r.getCurrentByte(); stopPred[c] == 1; {
		if c == '&' {
			c = r.getNextByte()
			switch c {
			// &amp; &apos;
			case 'a':
				if err := r.skipBytes(1); err == nil && bytes.HasPrefix(r.sliceToEnd(), []byte("mp;")) {
					r.position += 2
				} else if bytes.HasPrefix(r.sliceToEnd(), []byte("pos;")) {
					r.data[trail] = '\\' // overwrite
					r.position += 3
				}
			// &quot;
			case 'q':
				if err := r.skipBytes(1); err == nil && bytes.HasPrefix(r.sliceToEnd(), []byte("uot;")) {
					r.position += 3
				}
			// &gt;
			case 'g':
				if err := r.skipBytes(1); err == nil && bytes.HasPrefix(r.sliceToEnd(), []byte("t;")) {
					r.data[trail] = '>' // overwrite
					r.position++
				}
			// &lt;
			case 'l':
				if err := r.skipBytes(1); err == nil && bytes.HasPrefix(r.sliceToEnd(), []byte("t;")) {
					r.data[trail] = '<' // overwrite
					r.position++
				}
			default: // in case we cant find any entity
				trail++ // move after to r.position
			case 0:
				panic("end of file")
			}
			// &#...; - assumes ASCII -- not implemented
		} else if trail < r.position { // if tail is lagging the position, we meed to copy
			r.data[trail] = r.data[r.position]
		}
		if c = r.getNextByte(); c == 0 {
			return nil // error
		}
		trail++
	}
	return r.data[start:trail]
}

// parseAndAppendData adds a data node to the parent node.
func (r *RunXML) parseAndAppendData(parent *GenericNode) error {
	value := r.skipAndExpandCharacterRefs(lookupText, lookupTextPureNoWS)
	if value == nil {
		return fmt.Errorf("unable to append data node")
	}
	node := newNode(Data)
	node.Value = value
	parent.Value = value
	parent.AppendNode(node)
	//fmt.Println("adding datanode", node, node.Name, node.Value, string(node.Parent.Name))
	return nil
}

func (r *RunXML) contextError(v error) error {
	const contextSize = 40
	start := max(r.position-contextSize, 0)
	stop := min(r.position, len(r.data))
	leftcontext := r.data[start:stop]
	start = min(r.position+1, len(r.data))
	stop = min(r.position+contextSize, len(r.data))
	rightcontext := r.data[start:stop]
	if r.position > len(r.data)-1 {
		return fmt.Errorf("%v\n%v", v, string(leftcontext))
	}
	return fmt.Errorf("%v\n%v{%s}%v", v, string(leftcontext), string(r.getCurrentByte()),
		string(rightcontext))
}

// lookupNodeName - Node name (anything but space \n \r \t / > ? \0)
var lookupNodeName = &[256]byte{
	// 0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0, 1, 1, // 0
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 1
	0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, // 2
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, // 3
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 4
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 5
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 6
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 7
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 8
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 9
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // A
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // B
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // C
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // D
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // E
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // F
}

var lookupWhitespace = &[256]byte{
	// 0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 1, 0, 0, // 0
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 1
	1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 2
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 3
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 4
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 5
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 6
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 7
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 8
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 9
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // A
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // B
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // C
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // D
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // E
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // F
}

// Attribute name (anything but space \n \r \t / < > = ? ! \0)
var lookupAttributeName = &[256]byte{
	// 0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0, 1, 1, // 0
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 1
	0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, // 2
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, // 3
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 4
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 5
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 6
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 7
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 8
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 9
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // A
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // B
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // C
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // D
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // E
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // F
}

// Attribute data with single quote (anything but ' \0)
var lookupAttributeDataSQ = &[256]byte{
	// 0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 0
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 1
	1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, // 2
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 3
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 4
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 5
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 6
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 7
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 8
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 9
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // A
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // B
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // C
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // D
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // E
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // F
}

// Attribute data with single quote that does not require processing (anything but ' \0 &)
var lookupAttributeDataSQPure = &[256]byte{
	// 0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 0
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 1
	1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, // 2
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 3
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 4
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 5
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 6
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 7
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 8
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 9
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // A
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // B
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // C
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // D
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // E
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // F
}

// Attribute data with single quote that does not require processing (anything but " \0)
var lookupAttributeDataDQ = &[256]byte{
	// 0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 0
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 1
	1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 2
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 3
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 4
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 5
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 6
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 7
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 8
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 9
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // A
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // B
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // C
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // D
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // E
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // F
}

// Attribute data with double quote that does not require processing (anything but " \0 &)
var lookupAttributeDataDQPure = &[256]byte{
	// 0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 0
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 1
	1, 1, 0, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 2
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 3
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 4
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 5
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 6
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 7
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 8
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 9
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // A
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // B
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // C
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // D
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // E
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // F
}

// Text (i.e. PCDATA) that does not require processing when ws normalization is disabled
// (anything but < \0 &)
var lookupTextPureNoWS = &[256]byte{
	// 0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 0
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 1
	1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 2
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, // 3
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 4
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 5
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 6
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 7
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 8
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 9
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // A
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // B
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // C
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // D
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // E
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // F
}

// Text (i.e. PCDATA) (anything but < \0)
var lookupText = &[256]byte{
	// 0   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
	0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 0
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 1
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 2
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, // 3
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 4
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 5
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 6
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 7
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 8
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 9
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // A
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // B
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // C
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // D
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // E
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // F
}

func max(x, y int) int {
	if x >= y {
		return x
	}
	return y
}

func min(x, y int) int {
	if x <= y {
		return x
	}
	return y
}
