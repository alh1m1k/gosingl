package main

import (
	"errors"
	"go/ast"
	"strings"
)

var (
	UnknownTargetError = errors.New("target structure unknown")
)

type structFinder struct {
	name     string
	filename string
	s        *ast.TypeSpec
	m        []ast.Node
	i        []*ast.ImportSpec
}

func newStructFinder(name string, filename string) *structFinder {
	return &structFinder{name: strings.TrimSpace(name), filename: filename}
}

func (s *structFinder) find(n ast.Node) bool {

	switch lookedFor := n.(type) {
	case *ast.ImportSpec:
		s.i = append(s.i, lookedFor)
	case *ast.TypeSpec:
		if lookedFor.Name == nil {
			return true
		}
		if lookedFor.Name.Name == s.name {
			switch lookedFor.Type.(type) {
			case *ast.StructType:
				s.s = lookedFor
			case *ast.InterfaceType:
				s.s = lookedFor
			}
		}
	case *ast.FuncDecl:
		if lookedFor.Recv == nil {
			return false
		}
		switch exp := lookedFor.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			if exp.X.(*ast.Ident).Name == s.name {
				s.m = append(s.m, n)
			}
		case *ast.Ident:
			if exp.Name == s.name {
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

func (s *structFinder) imports() []*ast.ImportSpec {
	return s.i
}
