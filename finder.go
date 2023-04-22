package main

import (
	"context"
	"errors"
	"go/ast"
	"go/token"
	"io"
	"strings"

	"os"
)

type structFinder struct {
	name     string
	filename string
	s        *ast.TypeSpec
	m        []ast.Node
	i        []*ast.ImportSpec
}

func newStructFinder(name string, filename string) *structFinder {
	return &structFinder{name: name, filename: filename}
}

func (s *structFinder) find(n ast.Node) bool {

	switch lookedFor := n.(type) {
	case *ast.ImportSpec:
		s.i = append(s.i, lookedFor)
	case *ast.TypeSpec:
		if lookedFor.Name == nil {
			return true
		}
		if strings.Contains(lookedFor.Name.Name, s.name) {
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

func (s *structFinder) imports() []*ast.ImportSpec {
	return s.i
}

func generate(
	ctx context.Context,
	set *token.FileSet, //to context
	allFiles []*ast.File, //to context
	currentFile *ast.File,
	fileName string,
	dirname string,
	cfg Config,
) error {
	sf := newStructFinder(cfg.Structure, fileName)
	ast.Inspect(currentFile, sf.find)

	// structure not found
	if sf.structure() == nil {
		return errors.New("target structure not found")
	}

	var (
		resetFile *os.File
		writer    io.Writer
		err       error
	)

	g := newGenerator(sf.imports(), allFiles, set, dirname, cfg)
	if err = g.Do(ctx, sf.structure(), sf.methods()); err != nil {
		return err
	}

	if ctx.Value("writer") == nil {
		if !cfg.Write {
			writer = os.Stdout
			ctx = context.WithValue(ctx, "writer", writer)
		} else {
			resetFilePath := strings.Replace(fileName, ".go", "_singleton.go", 1)
			// delete if needed
			_ = os.Remove(resetFilePath)
			// writeType to a file
			resetFile, err = os.OpenFile(resetFilePath, os.O_CREATE|os.O_RDWR, 0600)
			if err != nil {
				return err
			}
			writer = resetFile
			ctx = context.WithValue(ctx, "writer", writer)
		}
	}

	if err = g.WriteTo(writer); err != nil {
		if resetFile != nil {
			os.Remove(resetFile.Name())
		}
		return err
	}
	return nil
}
