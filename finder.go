package main

import (
	"errors"
	"go/ast"
	"go/token"
	"strings"

	"io"
	"os"
)

type structFinder struct {
	name     string
	filename string
	s        *ast.TypeSpec
	m        []ast.Node
}

func newStructFinder(name string, filename string) *structFinder {
	return &structFinder{name: name, filename: filename}
}

func (s *structFinder) find(n ast.Node) bool {

	switch lookedFor := n.(type) {
	case *ast.TypeSpec:
		if lookedFor.Name == nil {
			return true
		}
		if strings.Contains(lookedFor.Name.Name, s.name) {
			if _, ok := lookedFor.Type.(*ast.StructType); ok {
				s.s = lookedFor
			}
		}
	case *ast.FuncDecl:
		if lookedFor.Recv == nil {
			return false
		}
		switch exp := lookedFor.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			if strings.Contains(exp.X.(*ast.Ident).Name, s.name) {
				s.m = append(s.m, n)
			}
		case *ast.Ident:
			if strings.Contains(exp.Name, s.name) {
				s.m = append(s.m, n)
			}
		}
	default:
		return true
	}

	return false
}

func (s *structFinder) structure() *ast.TypeSpec {
	return s.s
}

func (s *structFinder) methods() []ast.Node {
	return s.m
}

func generate(
	set *token.FileSet,
	currentFile *ast.File,
	allFiles []*ast.File,
	dirname string,
	pkgName string,
	fileName string,
	structToFind string,
	variableName string,
	write bool,
	customWriter io.Writer,
) error {
	sf := newStructFinder(structToFind, fileName)
	ast.Inspect(currentFile, sf.find)

	// structure not found
	if sf.structure() == nil {
		return errors.New("target structure not found")
	}

	var writer io.Writer
	if !write {
		if customWriter != nil {
			writer = customWriter
		} else {
			writer = os.Stdout
		}
	} else {
		resetFile := strings.Replace(fileName, ".go", "_singleton.go", 1)
		// delete if needed
		_ = os.Remove(resetFile)
		// writeType to a file
		w, err := os.OpenFile(resetFile, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return err
		}
		defer w.Close()
		writer = w
	}

	g := newGenerator(sf.structure(), sf.methods(), allFiles, set, dirname, pkgName, variableName, writer)
	return g.do()
}
