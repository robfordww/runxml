package runxml

import (
	"log"
	"testing"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func TestSimpleXML(t *testing.T) {
	// wrong start of xml
	xml := []byte("x<root><name></name></root>")
	r := NewDefaultRunXML()
	_, err := r.Parse(xml)
	if err == nil {
		t.Fatal("should fail")
	}
	t.Log(err)
}

func TestSimpleXML2(t *testing.T) {
	// wrong start of xml
	xml := []byte(`<dogregister version="1"> <dog><name alive='false'>Fido</name></dog> 
		<dog><name alive="true">Spike</name></dog>   </dogregister>`)
	r := NewDefaultRunXML()
	doc, err := r.Parse(xml)
	if err != nil {
		t.Fatal("should not fail", err)
	}
	docC := doc.CountChildren()
	if docC != 8 {
		t.Error("expected 5 children, found", docC)
	}
	for i := range doc.SendChildElements() {
		t.Log("\n", i.String(), "\n")
	}
}

func TestSendCloseChildren(t *testing.T) {
	xml := []byte(`<dogregister version="1"> <dog><name alive='false'>Fido</name></dog> 
		<dog><name alive="true">Spike</name></dog>   </dogregister>`)
	r := NewDefaultRunXML()
	doc, err := r.Parse(xml)
	if err != nil {
		t.Fatal("should not fail", err)
	}
	count := 0
	for i := range doc.SendCloseChildren() {
		t.Log("\n", i.String(), "\n")
		count++
	}
	if count != 1 {
		t.Error("wrong number of children")
	}
	count = 0
	for i := range doc.firstChild.SendCloseChildren() {
		t.Log("\n", i.String(), "\n")
		count++
	}
	if count != 2 {
		t.Error("wrong number of children")
	}
}

func TestFirstChildAndSiblings(t *testing.T) {
	xml := []byte(`<r><a>1</a>
		<b><b2>77</b2><b3>33</b3></b>
		<a>2</a></r>`)
	r := NewDefaultRunXML()
	doc, err := r.Parse(xml)
	if err != nil {
		t.Fatal("should not fail", err)
	}
	f := doc.GetFirstChild()
	if f == nil {
		t.Error("node should not be nil")
	}
	ns := f.GetNextSibling()
	if ns != nil {
		t.Log(f.String())
		t.Error("root should not have siblings")
	}
	ns = f.GetFirstChild()   // <a>
	ns = ns.GetNextSibling() // <b>
	if ns == nil {
		t.Error("next sibling should not be nil")
	}
	lc := ns.GetLastChild()
	if lc == nil {
		t.Error("last child returned nil")
	} else if string(lc.Value) != `33` {
		t.Error("last child value: expected 33 but found ", lc.Value)
	}
	ps := lc.GetPreviousSibling()
	if ps == nil {
		t.Error("expected previous sibling, but retuned nil")
	} else if string(ps.Value) != `77` {
		t.Error("previous sibling value: expected 77 but found ", lc.Value)
	}
	// test previous sibling, when it is the first sibling.
	if ps.GetPreviousSibling() != nil {
		t.Error("previous sibling shold be nil, when the caller is the first sibling")
	}
	ns = ns.GetNextSibling().GetNextSibling()
	if ns != nil {
		t.Error("next sibling should be nil")
	}
}
