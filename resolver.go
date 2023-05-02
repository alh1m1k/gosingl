package main

import (
	"fmt"
	"github.com/dave/jennifer/jen"
	"go/ast"
	"go/types"
	"log"
	"sync"
)

type pendingResolve struct {
	pkg   string
	ident *ast.Ident
	p     *jen.Statement
	o     types.Object
}

type resolveMap struct {
	index   []string
	resolve map[string]string
}
type mapResolveMap map[struct {
	Pkg, Target string
}]resolveMap

type Resolver interface {
	Resolve(ident *ast.Ident, pkg string, object types.Object) *jen.Statement
	Overlap(pkg, target string, resolve []string) //maybe []ast.Expr //currently too rough
	CompleteResolve(resolveMap resolveMap, allMaps mapResolveMap, group *sync.WaitGroup)
	NewResolver() Resolver
	Pop() Resolver
}

type genericResolver struct {
	pkg, target       string
	overlap           []string
	pending           []pendingResolve
	underlineResolver []Resolver
	parent            Resolver
}

func (r *genericResolver) Resolve(ident *ast.Ident, pkg string, object types.Object) *jen.Statement {
	statement := jen.Id(ident.Name)
	r.pending = append(r.pending, pendingResolve{
		ident: ident,
		pkg:   pkg,
		o:     object,
		p:     statement,
	})
	return statement
}

func (r *genericResolver) updateItems(resolveMap resolveMap, group *sync.WaitGroup) {
	defer group.Done()
	for _, statement := range r.pending {
		resetStatement(statement.p)
		if statement.ident.Obj == nil && ISScalarType(statement.ident.Name) { //it probably scalar // { //check Obj == nil is not enough
			statement.p.Id(statement.ident.Name)
		} else if resolved, ok := resolveMap.resolve[statement.ident.Name]; ok {
			if statement.ident.Obj == nil && ISScalarType(resolved) {
				statement.p.Id(resolved)
			} else {
				statement.p.Qual(statement.pkg, resolved)
			}
		} else if true { //local struct decl
			statement.p.Qual(statement.pkg, statement.ident.Name)
		} else { //for generics
			log.Println("WARNING: skip *ast.SelectorExpr(recursBuildParam) unsupported format")
		}
	}
}

func (r *genericResolver) CompleteResolve(rslMap resolveMap, allMaps mapResolveMap, parentGroup *sync.WaitGroup) {
	defer parentGroup.Done()
	group := &sync.WaitGroup{}
	group.Add(1 + len(r.underlineResolver))

	merge := rslMap
	if len(r.overlap) > 0 {
		//merge direction base type + var decl + user input
		merge = resolveMap{resolve: map[string]string{}, index: []string{}}
		if original, ok := allMaps[struct{ Pkg, Target string }{Pkg: r.pkg, Target: r.target}]; ok {
			if len(original.index) != len(r.overlap) {
				critical(fmt.Sprintf("type and impl are missmathed %s, %s", r.pkg, r.target))
			}
			for i, org := range original.index {
				if i < len(r.overlap) {
					merge.resolve[org] = r.overlap[i]
				} else {
					merge.resolve[org] = org
				}
				merge.index = append(merge.index, r.overlap[i])
			}
			for i, org := range merge.resolve {
				if overrided, ok := rslMap.resolve[org]; ok {
					merge.resolve[i] = overrided
				}
			}
		} else {
			critical(fmt.Sprintf("unable to find original %s, %s declaration", r.pkg, r.target))
		}
	}

	for i := range r.underlineResolver {
		go r.underlineResolver[i].CompleteResolve(merge, allMaps, group)
	}
	go r.updateItems(merge, group)

	group.Wait()
}

func (r *genericResolver) Pop() Resolver {
	return r.parent
}

func (r *genericResolver) Overlap(pkg, target string, resolve []string) {
	r.pkg, r.target = pkg, target
	r.overlap = resolve
}

func (r *genericResolver) NewResolver() Resolver {
	resolver := newResolver()
	r.underlineResolver = append(r.underlineResolver, resolver)
	resolver.parent = r
	return resolver
}

func newResolver() *genericResolver {
	resolver := &genericResolver{}
	return resolver
}
