package main

import (
	"context"
	"fmt"
	"github.com/dave/jennifer/jen"
	"go/ast"
	"log"
	"strings"
	"sync"
)

type Delayed interface {
	Do(ctx context.Context, target *ast.TypeSpec, targetMethods []ast.Node)
}

type VariableDecl struct {
	instance *jen.Statement
	target   *ast.TypeSpec
	Config
	vType
	resolved     []string
	resolveMap   resolveMap
	rootResolver Resolver
	processMut   sync.Mutex
}

func (v *VariableDecl) Declare() *jen.Statement {
	return v.instance
}

func (v *VariableDecl) Do(ctx context.Context, target *ast.TypeSpec, targetMethods []ast.Node) {
	v.processMut.Lock()
	defer v.processMut.Unlock()

	if v.vType == Auto {
		v.vType = v.updateVType(target, targetMethods)
	}

	if target != nil && target.Name.Name == v.Target {
		v.target = target
		if len(v.resolved) > 0 {
			if v.target.TypeParams == nil || v.target.TypeParams.List == nil {
				critical("target not a generic type")
			} else {
				v.resolveMap = make(map[string]string)
				total := 0
				maxLen := len(v.resolved)

				for i := range v.target.TypeParams.List {
					for _, ident := range v.target.TypeParams.List[i].Names {
						if total >= maxLen {
							total++
							continue
						}
						v.resolveMap[ident.Name] = v.resolved[total]
						total++
					}
				}
				if total != len(v.resolved) {
					critical("target generics count mismatch")
				}
			}
		}
	}
}

func (v *VariableDecl) CompleteResolve() {
	v.update()
	wait := &sync.WaitGroup{}
	wait.Add(1)
	go v.rootResolver.CompleteResolve(v.resolveMap, wait) //todo only if need
	wait.Wait()
}

func (v *VariableDecl) updateVType(target *ast.TypeSpec, targetMethods []ast.Node) vType {
	//auto
	if target != nil {
		switch target.Type.(type) {
		case *ast.InterfaceType:
			return Real
		case *ast.MapType:
			return Real
		case *ast.ArrayType:
			return Real
		case *ast.SliceExpr:
			return Real
		}
	}

	for i := range targetMethods {
		switch method := targetMethods[i].(type) {
		case *ast.FuncDecl:
			if method.Name.IsExported() {
				switch method.Recv.List[0].Type.(type) {
				case *ast.StarExpr: //mb [T]
					return Ref
				case *ast.Ident:
					return Real
				default:
					log.Println("wrong declaration attempt")
				}
			}
		}
	}

	return Auto
}

func (v *VariableDecl) update() {
	resetStatement(v.instance)
	switch v.vType {
	case Real:
		v.instance.Id(v.Variable).Qual(v.Package, v.Target)
	case Ref:
		v.instance.Id(v.Variable).Op("*").Qual(v.Package, v.Target)
	default:
		v.instance.Id(v.Variable).Op("*").Qual(v.Package, v.Target)
	}
	if len(v.resolved) > 0 {
		v.instance.TypesFunc(func(group *jen.Group) {
			for _, part := range v.resolved {
				if strings.Contains(part, ".") {
					qual := strings.Split(part, ".")
					if len(qual) == 2 {
						group.Add(jen.Qual(qual[0], qual[1]))
					} else {
						critical(fmt.Sprintf("WARNING: probably incorrect generic resolve %s", part))
						group.Add(jen.Id(part))
					}
				} else {
					group.Add(jen.Id(part))
				}
			}
		})
	} else {
		//log.Println("empty resolver")
	}
}

func NewVariableDecl(cfg Config, vType vType, resolved []string) *VariableDecl {
	return &VariableDecl{
		instance:     &jen.Statement{},
		Config:       cfg,
		vType:        vType,
		resolved:     resolved,
		rootResolver: newResolver(),
	}
}

func NewVariableDeclFromConfig(cfg Config) *VariableDecl {
	//log.Println(cfg.Variable)
	generics := resolveGenerics2(cfg)
	if strings.Index(cfg.Variable, "*") == 0 {
		cfg.Variable = clearVarFromDeclaration(cfg.Variable)
		return NewVariableDecl(cfg, Real, generics)
	} else if strings.Index(cfg.Variable, "&") == 0 {
		cfg.Variable = clearVarFromDeclaration(cfg.Variable)
		return NewVariableDecl(cfg, Ref, generics)
	} else {
		//auto
		cfg.Variable = clearVarFromDeclaration(cfg.Variable)
		return NewVariableDecl(cfg, Auto, generics)
	}
}

// todo refactor
func resolveGenerics2(cfg Config) []string {
	start, end := strings.Index(cfg.Variable, "["), strings.Index(cfg.Variable, "]")
	if start < 0 || end < 0 || end-start <= 1 {
		return []string{}
	}
	return strings.Split(cfg.Variable[start+1:end], ",")
}
