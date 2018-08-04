package runxml_test

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/robfordww/runxml"
)

func openLargeTestFile(t *testing.T) *runxml.GenericNode {
	rx := runxml.NewDefaultRunXML()
	//f, err := os.Open("../../xmltestfiles/enwiki_short.xml.gz")
	f, err := os.Open("../../xmltestfiles/enwiki-20180220-pages-logging20.xml.gz")
	defer f.Close()
	if err != nil {
		t.Fatal(err)
	}
	gr, err := gzip.NewReader(f)
	defer gr.Close()
	if err != nil {
		t.Fatal(err)
	}
	b, err := ioutil.ReadAll(gr)
	t.Log("Size of data:", len(b)/1E6, "MB")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Parse start", time.Now())
	documentNode, err := rx.Parse(b)
	fmt.Println("Parse stop", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return documentNode
}

func TestWikipediaLargeXML(t *testing.T) {
	documentNode := openLargeTestFile(t)
	root := documentNode.GetFirstChild()
	ch := root.SendCloseChildren()

	// Simulated generated code
	l := LogItems{}
	l.MapXML(ch)

	fmt.Print(root.CountChildren())
	//spew.Dump(l)
}

// LogItems <-- <logitem>
type LogItems []LogItem

// Arraymapping
func (l *LogItems) MapXML(ch chan *runxml.GenericNode) {
	for n := range ch {
		//spew.Dump(n)
		if string(n.Name) == "logitem" {
			// Find first relevant element
			// If target is array, pass the channel to its method, and add the result
			item := LogItem{}
			item.MapXML(n.SendCloseChildren())
			*l = append(*l, item)
			// If target
		}
	}
}

type LogItem struct {
	Id          int
	Timestamp   time.Time
	Comment     string
	Typename    string
	Logtitle    string
	Contributor Contributor
}

func (l *LogItem) MapXML(ch chan *runxml.GenericNode) {
	// continue reading the channel
	for j := range ch { // continue
		//fmt.Print(string(j.Name), "-", j.NodeType, "\n")
		switch string(j.Name) {
		case "id":
			v, _ := strconv.Atoi(string(j.Value))
			l.Id = v
		case "timestamp":
			p, _ := time.Parse(string(j.Value), time.RFC3339Nano)
			l.Timestamp = p
		case "comment":
			l.Comment = string(j.Value)
		case "type":
			l.Typename = string(j.Value)
		case "logtitle":
			l.Logtitle = string(j.Value)
		case "contributor":
			l.Contributor.MapXML(j.SendCloseChildren())
		case "logitem":
			//fmt.Println("end of tag")
			return
		}
	}
	//logitems = append(logitems, l)
}

type Contributor struct {
	Username string
	Id       string
}

func (c *Contributor) MapXML(ch chan *runxml.GenericNode) {
	// continue reading the channel
	for j := range ch { // continue
		//fmt.Print(string(j.Name), "/", j.NodeType, "\n")
		if j == nil {
			panic("nil pointer")
		}
		switch string(j.Name) {
		case "username":
			c.Username = string(j.Value)
		case "id":
			c.Id = string(j.Value)
		case "contributor":
			return
		}
	}
	//logitems = append(logitems, l)
}

// <logitem>
// <id>62809477</id>
// <timestamp>2015-02-27T03:27:44Z</timestamp>
// <contributor>
//   <username>ClueBot NG</username>
//   <id>13286072</id>
// </contributor>
// <comment>automatic</comment>
// <type>review</type>
// <action>approve-a</action>
// <logtitle>Jamie Colby</logtitle>
// <params xml:space="preserve">649036197
// 647791618
// 20150227032744</params>
// </logitem>
