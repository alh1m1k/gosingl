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
	Do(ctx context.Context, target *ast.TypeSpec, targetMethods []ast.Node, cfg Config)
}

type VariableDecl struct {
	instance *jen.Statement
	target   *ast.TypeSpec
	Config
	vType
	resolved     []string
	rootResolver Resolver
	processMut   sync.Mutex
	generics     map[struct {
		Pkg, Target string
	}]resolveMap
}

func (v *VariableDecl) Declare() *jen.Statement {
	return v.instance
}

func (v *VariableDecl) Do(ctx context.Context, target *ast.TypeSpec, targetMethods []ast.Node, cfg Config) {
	v.processMut.Lock()
	defer v.processMut.Unlock()

	if v.vType == Auto && cfg.Target == v.Target {
		v.vType = v.updateVType(target, targetMethods)
	}

	if target != nil {

		if cfg.Target == v.Target {
			v.target = target
		}

		if len(v.resolved) > 0 {
			if target.TypeParams == nil || target.TypeParams.List == nil {
				if target.Name.Name == v.Target {
					critical("target not a generic type")
				}
			} else {
				rslMap := resolveMap{resolve: map[string]string{}, index: []string{}}
				total := 0
				maxLen := len(v.resolved)

				for i := range target.TypeParams.List {
					for _, ident := range target.TypeParams.List[i].Names {
						if total >= maxLen {
							total++
							continue
						}
						rslMap.index = append(rslMap.index, ident.Name)
						rslMap.resolve[ident.Name] = v.resolved[total] //if v.Target != cfg.Target this line doesn't make sense.
						//it will be recalculated later in proper context
						total++
					}
				}
				v.generics[struct{ Pkg, Target string }{Pkg: cfg.Package, Target: cfg.Target}] = rslMap
				if total != len(v.resolved) {
					critical(fmt.Sprintf("%s target generics count mismatch", target.Name))
				}
			}
		}
	}
}

func (v *VariableDecl) CompleteResolve() {
	v.update()
	wait := &sync.WaitGroup{}
	wait.Add(1)
	go v.rootResolver.CompleteResolve(v.generics[struct{ Pkg, Target string }{Pkg: v.Package, Target: v.Target}], v.generics, wait) //todo only if need
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
		generics:     map[struct{ Pkg, Target string }]resolveMap{},
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
