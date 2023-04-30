package main

import (
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

type resolveMap map[string]string

type Resolver interface {
	Resolve(ident *ast.Ident, pkg string, object types.Object) *jen.Statement
	CompleteResolve(resolveMap resolveMap, group *sync.WaitGroup)
	NewResolver() Resolver
}

type genericResolver struct {
	id                int64
	pending           []pendingResolve
	underlineResolver []Resolver
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
		} else if resolved, ok := resolveMap[statement.ident.Name]; ok {
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

func (r *genericResolver) CompleteResolve(resolveMap resolveMap, parentGroup *sync.WaitGroup) {
	defer parentGroup.Done()
	group := &sync.WaitGroup{}
	group.Add(1 + len(r.underlineResolver))

	for i := range r.underlineResolver {
		go r.underlineResolver[i].CompleteResolve(resolveMap, group)
	}
	go r.updateItems(resolveMap, group)

	group.Wait()
}

func (r *genericResolver) NewResolver() Resolver {
	resolver := newResolver()
	r.underlineResolver = append(r.underlineResolver, resolver)
	return resolver
}

func newResolver() Resolver {
	resolver := &genericResolver{}
	return resolver
}
