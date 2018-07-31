package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"strings"
)

// File holds a single parsed file and associated data.
type File struct {
	pkg  *Package  // Package to which this file belongs.
	file *ast.File // Parsed AST.
	// These fields are reset for each type being generated.
	typeName string // Name of the constant type.
	//values   []Value // Accumulator for constant values of that type.

	trimPrefix  string
	lineComment bool
}

type Package struct {
	dir      string
	name     string
	defs     map[*ast.Ident]types.Object
	files    []*File
	typesPkg *types.Package
}

// Generator holds the state of the analysis. Primarily used to buffer
// the output for format.Source.
type Generator struct {
	buf bytes.Buffer // Accumulated output.
	pkg *Package     // Package we are scanning.

	trimPrefix  string
	lineComment bool
}

// parsePackageFiles parses the package occupying the named files.
func (g *Generator) parsePackageFiles(names []string) {
	g.parsePackage(".", names, nil)
}

// Printf adds a formated string to buffer
func (g *Generator) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

// parsePackage analyzes the single package constructed from the named files.
// If text is non-nil, it is a string to be used instead of the content of the file,
// to be used for testing. parsePackage exits if there is an error.
func (g *Generator) parsePackage(directory string, names []string, text interface{}) {
	var files []*File
	var astFiles []*ast.File
	g.pkg = new(Package)
	fs := token.NewFileSet()
	for _, name := range names {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		parsedFile, err := parser.ParseFile(fs, name, text, parser.ParseComments)
		if err != nil {
			log.Fatalf("parsing package: %s: %s", name, err)
		}
		astFiles = append(astFiles, parsedFile)
		files = append(files, &File{
			file:        parsedFile,
			pkg:         g.pkg,
			trimPrefix:  g.trimPrefix,
			lineComment: g.lineComment,
		})
	}
	if len(astFiles) == 0 {
		log.Fatalf("%s: no buildable Go files", directory)
	}
	g.pkg.name = astFiles[0].Name.Name
	g.pkg.files = files
	g.pkg.dir = directory
	//g.pkg.typeCheck(fs, astFiles)
}

// parsePackageDir parses the package residing in the directory.
func (g *Generator) parsePackageDir(directory string) {
	bd := build.Default
	pkg, err := bd.ImportDir(directory, 0)
	if err != nil {
		log.Fatalf("cannot process directory %s: %s", directory, err)
	}
	var names []string
	names = append(names, pkg.GoFiles...)
	names = append(names, pkg.CgoFiles...)
	// TODO: Need to think about constants in test files. Maybe write type_string_test.go
	// in a separate pass? For later.
	// names = append(names, pkg.TestGoFiles...) // These are also in the "foo" package.
	names = append(names, pkg.SFiles...)
	names = prefixDirectory(directory, names)
	g.parsePackage(directory, names, nil)
}

// format returns the gofmt-ed contents of the Generator's buffer.
func (g *Generator) format() []byte {
	src, err := format.Source(g.buf.Bytes())
	if err != nil {
		// Should never happen, but can arise when developing this code.
		// The user can compile the output to see the error.
		log.Printf("warning: internal error: invalid Go generated: %s", err)
		log.Printf("warning: compile the package to analyze the error")
		return g.buf.Bytes()
	}
	return src
}

func (g *Generator) generate(typeName string) {
	//stub
}
