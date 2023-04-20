package main

import (
	"context"
	"errors"
	"go/ast"
	"go/token"
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
			if _, ok := lookedFor.Type.(*ast.StructType); ok {
				//log.Println("find")
				s.s = lookedFor
			} else {
				//log.Println("w type", reflect.ValueOf(lookedFor).String())
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
		err       error
	)

	if ctx.Value("writer") == nil {
		if !cfg.Write {
			ctx = context.WithValue(ctx, "writer", os.Stdout)
		} else {
			resetFilePath := strings.Replace(fileName, ".go", "_singleton.go", 1)
			// delete if needed
			_ = os.Remove(resetFilePath)
			// writeType to a file
			resetFile, err = os.OpenFile(resetFilePath, os.O_CREATE|os.O_RDWR, 0600)
			if err != nil {
				return err
			}
			ctx = context.WithValue(ctx, "writer", resetFile)
		}
	}

	g := newGenerator(sf.imports(), sf.structure(), sf.methods(), allFiles, set, dirname, cfg)
	if err = g.do(ctx); err != nil {
		if resetFile != nil {
			resetFile.Close()
			_ = os.Remove(resetFile.Name())
		}
		return err
	} else {
		if resetFile != nil {
			resetFile.Close()
		}
		return nil
	}
}
