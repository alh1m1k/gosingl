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
				for i := range target.TypeParams.List {
					for _, ident := range target.TypeParams.List[i].Names {
						rslMap.index = append(rslMap.index, ident.Name)
						total++
					}
				}
				v.generics[struct{ Pkg, Target string }{Pkg: cfg.Package, Target: cfg.Target}] = rslMap
			}
		}
	}
}

func (v *VariableDecl) CompleteResolve() {
	v.update()
	wait := &sync.WaitGroup{}
	wait.Add(1)
	if len(v.resolved) > 0 {
		genericsRoot := v.generics[struct{ Pkg, Target string }{Pkg: v.Package, Target: v.Target}]
		if len(genericsRoot.index) != len(v.resolved) {
			critical(fmt.Sprintf("%s target generics count mismatch %d:%d", v.Target, len(genericsRoot.index), len(v.resolved)))
		}
		for i := range genericsRoot.index {
			genericsRoot.resolve[genericsRoot.index[i]] = v.resolved[i]
		}
	}
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

	var resolveVType func(exp ast.Expr) vType
	resolveVType = func(exp ast.Expr) vType {
		switch newT := exp.(type) {
		case *ast.StarExpr: //mb [T]
			return Ref
		case *ast.Ident:
			return Real
		case *ast.IndexListExpr:
			return resolveVType(newT.X)
		default:
			log.Println("wrong declaration attempt")
		}
		return Auto
	}

	for i := range targetMethods {
		switch method := targetMethods[i].(type) {
		case *ast.FuncDecl:
			if method.Name.IsExported() {
				if rsl := resolveVType(method.Recv.List[0].Type); rsl != Auto {
					return rsl
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
			for _, resolve := range v.resolved {
				var (
					statement *jen.Statement = &jen.Statement{}
				)
				if strings.Index(resolve, "*") == 0 {
					statement.Op("*")
					if len(resolve) > 1 {
						resolve = strings.TrimSpace(resolve[1:]) //drop symbol *
					}
				}
				if strings.Contains(resolve, ".") {
					qual := strings.Split(resolve, ".")
					if len(qual) == 2 {
						qual[0], qual[1] = strings.TrimSpace(qual[0]), strings.TrimSpace(qual[1])
						group.Add(statement.Qual(qual[0], qual[1]))
					} else {
						critical(fmt.Sprintf("WARNING: probably incorrect generic resolve %s", resolve))
						group.Add(statement.Id(resolve))
					}
				} else {
					group.Add(statement.Id(resolve))
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
	result := strings.Split(cfg.Variable[start+1:end], ",")
	for i := range result {
		result[i] = strings.TrimSpace(result[i])
	}
	return result
}
