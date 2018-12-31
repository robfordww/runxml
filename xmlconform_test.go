package runxml

import (
	"path/filepath"
	"testing"
)

func TestInvalideFiles(t *testing.T) {
	r := NewDefaultRunXML()
	// Not all tests are run; might be something that can be addressed later
	testdir := "xmltestfiles/xmlconf/xmltest/not-wf/sa/0[0-3]*.xml"
	f, err := filepath.Glob(testdir)
	if err != nil {
		t.Fatal(err)
	}
	excludeList := map[string]bool{
		"002.xml": true, // <.doc></.doc>
		"007.xml": true, // <doc>&amp no refc</doc>
		"008.xml": true, // <doc>&.entity;</doc>
		"009.xml": true, // <doc>&#RE;</doc>
		"010.xml": true, // <doc>A & B</doc>
		"014.xml": true, // <doc a1="<foo>"></doc>
		"020.xml": true, // <doc a1="A & B"></doc>
		"021.xml": true, // <doc a1="a&b"></doc>
		"022.xml": true, // <doc a1="&#123:"></doc>
		"023.xml": true, // <doc 12="34"></doc>
		"024.xml": true, // <123></123>
		"025.xml": true, // <doc>]]></doc>
		"026.xml": true, // <doc>]]]></doc>
		"029.xml": true, // <doc>abc]]]>def</doc>
		"030.xml": true, // <doc>A form feed () is not legal in data</doc>
		"031.xml": true, // <doc><?pi a form feed () is not allowed in a pi?></doc>
		"032.xml": true, // <doc><!-- a form feed () is not allowed in a comment --></doc>
		"033.xml": true, // <doc>abcdef</doc>
		"034.xml": true, // <doc>A form-feed is not white space or a name character</doc>
		"038.xml": true, // <doc x="foo" y="bar" x="baz"></doc>
	}
	testhelp(t, r, f, excludeList, false)
}

type testDir struct {
	path      string
	exclusion map[string]bool
}

var testDirs = []testDir{
	/*testDir{
		path:      "xmltestfiles/xmlconf/eduni/errata-2e/*.xml",
		exclusion: map[string]bool{},
	},
	testDir{
		path:      "xmltestfiles/xmlconf/eduni/errata-3e/*.xml",
		exclusion: map[string]bool{},
	},
	testDir{
		path:      "xmltestfiles/xmlconf/eduni/errata-4e/*.xml",
		exclusion: map[string]bool{},
	},*/
	testDir{
		path: "xmltestfiles/xmlconf/ibm/xml-1.1/valid/*/*.xml",
		exclusion: map[string]bool{"p49pass1.xml": true,
			"p50pass1.xml": true,
		},
	},
	testDir{
		path: "xmltestfiles/xmlconf/ibm/valid/*/*.xml",
		exclusion: map[string]bool{"p49pass1.xml": true,
			"p50pass1.xml": true,
		},
	},
	testDir{
		path: "xmltestfiles/xmlconf/oasis/*pass*.xml",
		exclusion: map[string]bool{"p49pass1.xml": true,
			"p50pass1.xml": true,
		},
	},
	testDir{
		path:      "xmltestfiles/xmlconf/xmltest/valid/sa/*.xml",
		exclusion: map[string]bool{},
	},
	testDir{
		path:      "xmltestfiles/xmlconf/xmltest/valid/not-sa/*.xml",
		exclusion: map[string]bool{},
	},
	testDir{
		path:      "xmltestfiles/xmlconf/xmltest/valid/ext-sa/*.xml",
		exclusion: map[string]bool{},
	},
	testDir{
		path:      "xmltestfiles/xmlconf/sun/valid/*.xml",
		exclusion: map[string]bool{},
	},
}

func TestValidFiles(t *testing.T) {
	numFiles := 0
	for i := range testDirs {
		r := NewDefaultRunXML()
		f, err := filepath.Glob(testDirs[i].path)
		if err != nil {
			t.Fatal(err)
		}
		excludeList := testDirs[i].exclusion
		testhelp(t, r, f, excludeList, true)
		numFiles += len(f) - len(excludeList)
	}
	t.Log("Files tested", numFiles)
}

func testhelp(t *testing.T, r *RunXML, f []string, excludeList map[string]bool, expectSuccess bool) {
	var filesToParse []string
	for i := range f {
		_, fname := filepath.Split(f[i])
		if _, ok := excludeList[fname]; !ok {
			filesToParse = append(filesToParse, f[i])
		} else {
			t.Log("Excluded", fname)
		}
	}

	for _, fn := range filesToParse {
		//t.Log("Parsing", fn) // enable on debug
		pf, err := r.ParseFile(fn)
		if err == nil && !expectSuccess {
			pf.PrintChildren()
			t.Log(fn, "expected fail, but test succeded")
			t.FailNow()
		} else if err != nil && expectSuccess {
			t.Log(fn, "expected success, but test failed:", err)
			pf.PrintChildren()
			t.FailNow()
		}
	}
}
