package main

import (
	"errors"
	"fmt"
	"go/ast"
	"strings"
)

var (
	UnknownTargetError = errors.New("target Structure unknown")
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

func (s *structFinder) Find(n ast.Node) bool {

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
			case *ast.SliceExpr:
				s.s = lookedFor
			case *ast.ArrayType:
				s.s = lookedFor
			case *ast.MapType:
				s.s = lookedFor
			}
		}
	case *ast.FuncDecl:
		if lookedFor.Recv == nil {
			return false
		}
		s.recursiveResolveRcv(lookedFor, lookedFor.Recv.List[0].Type)
	default:

		return true
	}

	return false
}

func (s *structFinder) recursiveResolveRcv(n ast.Node, t ast.Expr) {
	switch exp := t.(type) {
	case *ast.StarExpr:
		s.recursiveResolveRcv(n, exp.X)
	case *ast.Ident:
		if exp.Name == s.name {
			s.m = append(s.m, n)
		}
	case *ast.IndexListExpr:
		s.recursiveResolveRcv(n, exp.X)
	default:
		critical(fmt.Sprintf("WARNING: abnormal func rcv: %typ", n.(*ast.FuncDecl).Recv.List[0].Type))
	}
}

func (s *structFinder) Structure() *ast.TypeSpec {
	return s.s
}

func (s *structFinder) Methods() []ast.Node {
	return s.m
}

func (s *structFinder) Imports() []*ast.ImportSpec {
	return s.i
}

type pathFinder struct {
	pkg, target string
	Success     bool
	ast.Node
}

func newPathFinder(currentPkg string) *pathFinder {
	return &pathFinder{pkg: currentPkg}
}

func (s *pathFinder) Find(n ast.Node) bool {
	switch lookedFor := n.(type) {
	case *ast.SelectorExpr:
		s.pkg = lookedFor.X.(*ast.Ident).Name
		s.target = lookedFor.Sel.Name
		s.Node = lookedFor
		s.Success = true
		return false
	case *ast.Ident:
		s.target = lookedFor.Name
		s.Node = lookedFor
		s.Success = true
		return false
	}
	return true
}

func (s *pathFinder) Structure() ast.Node {
	return s.Node
}

func (s *pathFinder) Path() (pkg, target string) {
	return s.pkg, s.target
}
