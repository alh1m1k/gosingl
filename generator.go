package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/dave/jennifer/jen"
	"go/ast"
	"go/types"
	"log"
	"reflect"
	"strings"
)

var ParserWarning = errors.New("WARNING:")

type loaderCallback func(ctx context.Context, cfg Config) error

type wrappedFunctionDeclaration struct {
	Name        string
	Content     []*jen.Statement
	IsInterface bool
	Signature   types.Object
	Config
}

// generator will work on the selected Structure of one file
type generator struct {
	defs            packageDefs
	pkg             *types.Package
	imports         []*ast.ImportSpec
	path            string
	cfg             Config
	loader          loaderCallback
	resolver        Resolver
	deep, generated int
	output          []*wrappedFunctionDeclaration
}

func newGenerator(imports []*ast.ImportSpec, cfg Config, loader loaderCallback, defs packageDefs, path string) *generator {
	return &generator{
		defs:    defs,
		imports: imports,
		path:    path,
		cfg:     cfg,
		loader:  loader,
		output:  nil,
	}
}

func (g *generator) Result() []*wrappedFunctionDeclaration {
	return g.output
}

func (g *generator) Do(ctx context.Context, target *ast.TypeSpec, targetMethods []ast.Node) (context.Context, error) {

	if target != nil {
		switch target.Type.(type) {
		case *ast.StructType:
			//nope
		case *ast.InterfaceType:
			//nope
		case *ast.MapType:
			//nope
		case *ast.SliceExpr:
			//nope
		case *ast.ArrayType:
			//nope
		default:
			return ctx, UnknownTargetError
		}
	}

	if g.output == nil {
		g.output = bufferFrom(ctx, g.cfg)
	}

	//chaining resolvers
	g.resolver = resolverFrom(ctx).NewResolver()

	//todo struct or bitmap
	g.deep = deepFrom(ctx)

	for i := range targetMethods {
		switch method := targetMethods[i].(type) {
		case *ast.FuncDecl:
			if method.Name.IsExported() /*&& g.checker.Check(method.Name.Name, method.Type.Params, method.Type.Results, g.interfaceWalk, g.cfg)*/ {
				g.output = append(g.output, g.wrapFunction(ctx, method.Name, method.Type.Params, method.Type.Results, method.Doc.Text()))
			}
		default:
			log.Println("ubnormal value in targetMethods", reflect.ValueOf(targetMethods[i]).String())
		}
	}

	if target != nil {
		switch structure := target.Type.(type) {
		case *ast.StructType:
			for _, field := range structure.Fields.List {
				if g.isIgnored(field.Tag) {
					continue
				}
				if err := g.digField(ctx, field, field.Type); err != nil {
					if errors.Is(err, ParserWarning) {
						log.Println(err)
						continue
					} else {
						return ctx, err
					}
				}
			}
		case *ast.InterfaceType:
			ctx = withInterfaceWalk(ctx)
			for _, field := range structure.Methods.List {
				if g.isIgnored(field.Tag) {
					continue
				}
				if err := g.digField(ctx, field, field.Type); err != nil {
					if errors.Is(err, ParserWarning) {
						log.Println(err)
						continue
					} else {
						return ctx, err
					}
				}
			}
		case *ast.MapType:
			//no internal field to inspect
		case *ast.SliceExpr:
			//no internal field to inspect
		case *ast.ArrayType:
			//no internal field to inspect
		default:
			panic("must not happened")
		}
	}

	delayed := pendingStatementFrom(ctx)
	for _, d := range delayed {
		d.Do(ctx, target, targetMethods, g.cfg)
	}

	ctx = withCallPrefix(ctx, []string{}) //call complete, reset prefix

	return ctx, nil
}

// scan target field
func (g *generator) digField(ctx context.Context, field *ast.Field, inner ast.Expr) error {
	switch fieldTyped := inner.(type) {
	case *ast.FuncType:
		for i := range field.Names {
			if field.Names[i].IsExported() && len(field.Names[i].Name) > 0 {
				g.output = append(g.output, g.wrapFunction(ctx, field.Names[i], fieldTyped.Params, fieldTyped.Results, field.Doc.Text()))
			}
		}
	case *ast.StarExpr:
		g.digField(ctx, field, fieldTyped.X) //keep *
	case *ast.SelectorExpr:
		if len(field.Names) == 0 {
			//this !interfaceWalkFrom ... withCallPrefix
			//setup callPrefix as current non interface type or root interface type
			//in interfaceWalk
			if !interfaceWalkFrom(ctx) {
				ctx = withCallPrefix(ctx, append(callPrefixFrom(ctx), fieldTyped.Sel.Name))
			}
			if err := g.digExternalDecl(ctx, fieldTyped); err != nil {
				return err
			}
		}
	case *ast.Ident:
		if len(field.Names) == 0 {
			//todo check with map[ident]decl
			if !interfaceWalkFrom(ctx) {
				ctx = withCallPrefix(ctx, append(callPrefixFrom(ctx), fieldTyped.Name))
			}
			if fieldTyped.Obj != nil && fieldTyped.Obj.Kind == ast.Typ {
				switch fieldTyped.Obj.Decl.(*ast.TypeSpec).Type.(type) {
				case *ast.StructType:
					err := g.parsePackage(ctx, g.cfg.Package, fieldTyped.Name, fmt.Sprintf("<%s>", field.Type))
					if err != nil {
						return err
					}
				case *ast.InterfaceType:
					ctx = withInterfaceWalk(ctx)
					err := g.parsePackage(ctx, g.cfg.Package, fieldTyped.Name, fmt.Sprintf("<%s>", field.Type))
					if err != nil {
						return err
					}
					//todo inline interface decl
				}
			}
		}
	case *ast.InterfaceType:
		if len(field.Names) == 0 {
			ctx = withInterfaceWalk(ctx)
			for _, field := range fieldTyped.Methods.List {
				if g.isIgnored(field.Tag) {
					continue
				}
				if err := g.digField(ctx, field, field.Type); err != nil {
					if errors.Is(err, ParserWarning) {
						info(err)
						continue
					} else {
						//interface dig interrupt with error
						return err
					}
				}
			}
		}
	case *ast.IndexListExpr:
		//new resolver context as it was new generic
		g.resolver = resolverFrom(ctx).NewResolver()

		if len(fieldTyped.Indices) > 0 {
			newMapping := make([]string, 0, len(fieldTyped.Indices))
			for i := range fieldTyped.Indices {
				newMapping = append(newMapping, fieldTyped.Indices[i].(*ast.Ident).Name)
			}
			pf := newPathFinder(g.cfg.Package)
			ast.Inspect(fieldTyped.X, pf.Find)
			if pf.Success {
				pkg, target := pf.Path()
				g.resolver.Overlap(pkg, target, newMapping)
			}
		}
		err := g.digField(ctx, field, fieldTyped.X)

		g.resolver = g.resolver.Pop()

		return err
	default:
		//
	}
	return nil
}

func (g *generator) digExternalDecl(ctx context.Context, str *ast.SelectorExpr) error {
	if strIdn, ok := str.X.(*ast.Ident); ok {
		importSpec := g.localeImport(strIdn.Name)
		if importSpec == nil {
			return fmt.Errorf("%w unable to locate import by name", ParserWarning)
		}
		err := g.parsePackage(ctx, importCanon(importSpec.Path.Value), str.Sel.Name, fmt.Sprintf("<%s.%s>", str.X, str.Sel.Name))
		if err != nil {
			return err
		}
		return nil
	} else {
		return fmt.Errorf("%w skip *ast.SelectorExpr(do) unsupported format %v", ParserWarning, str.X)
	}
}

func (g *generator) parsePackage(ctx context.Context, pkg, structure, comment string) error {

	ctx = withResolver(ctx, g.resolver)
	ctx = withIncDeep(ctx)
	cfg := g.cfg

	if g.cfg.Deep > 0 {
		cfg.Deep--
	}

	cfg.Package = importCanon(pkg)
	cfg.Target = structure
	cfg.Comment = comment

	err := g.loader(ctx, cfg)
	if err != nil {
		return err
	}
	return nil
}

func (g *generator) isIgnored(tag *ast.BasicLit) bool {
	if tag == nil {
		return false
	}
	bst := reflect.StructTag(strings.Trim(tag.Value, "`"))
	if tc := bst.Get("singl"); tc == "ignore" {
		return true
	}
	return false
}

func (g *generator) wrapFunction(ctx context.Context, ident *ast.Ident, in, out *ast.FieldList, comment string) *wrappedFunctionDeclaration {

	decl := &wrappedFunctionDeclaration{
		Name:        ident.Name,
		Content:     make([]*jen.Statement, 0),
		IsInterface: false,
		Signature:   nil,
		Config:      g.cfg,
	}

	if len(comment) > 0 {
		decl.Content = append(decl.Content, jen.Comment(comment))
	}

	namer := namerFrom(ctx).NewNamer() //refresh namer for every 1 level function
	fnBuilder := jen.Add(g.buildFunction(ident.Name, in, out, namer, false))
	underFn := jen.Id(g.cfg.Variable)

	//prefix is callContext ie Instance.[callCtx[0]...callCtx[n]].DoSome()
	for _, prefix := range callPrefixFrom(ctx) {
		underFn.Dot(prefix)
	}

	underFn.Dot(ident.Name)
	underFn.CallFunc(func(group *jen.Group) {
		names := namer.Values()
		for _, field := range in.List {
			for _, fieldIdent := range field.Names {
				if g.isAnonIdent(fieldIdent) {
					g.addFnParam(group, field, names[0]) //use first in stack namer value
					names = names[1:]
				} else {
					g.addFnParam(group, field, fieldIdent.Name)
				}
			}
			if len(field.Names) == 0 {
				g.addFnParam(group, field, names[0])
				names = names[1:]
			}
		}
	})
	fnBuilder.BlockFunc(func(grp *jen.Group) {
		if out != nil && out.NumFields() > 0 {
			grp.Return(underFn)
		} else {
			grp.Add(underFn)
		}
	})

	g.generated++

	decl.Content = append(decl.Content, fnBuilder)
	decl.IsInterface = interfaceWalkFrom(ctx)
	decl.Signature = g.defs[ident]

	return decl
}

func (g *generator) addFnParam(fnGroup *jen.Group, field *ast.Field, name string) {
	if _, ok := field.Type.(*ast.Ellipsis); ok {
		fnGroup.Add(jen.Id(name).Op("..."))
	} else {
		fnGroup.Add(jen.Id(name))
	}
}

func (g *generator) isAnonIdent(ident *ast.Ident) bool {
	if ident == nil {
		return true
	}
	if ident.Name == "" || ident.Name == "_" {
		return true
	}
	return false
}

func (g *generator) buildFunction(name string, in, out *ast.FieldList, namer Namer, buildSignature bool) *jen.Statement {
	fnBuilder := jen.Func()
	if len(name) > 0 {
		fnBuilder.Id(name)
	}
	if in.NumFields() > 0 {
		fnBuilder.Params(g.buildParams(in, namer, buildSignature)...)
	} else {
		fnBuilder.Op("()")
	}
	outFieldsCnt := out.NumFields()
	if out != nil && out.NumFields() > 0 {
		if outFieldsCnt == 1 && out.List[0].Names == nil {
			fnBuilder.List(g.buildParams(out, nil, buildSignature)...)
		} else {
			// (named type) as ret value must be in bracket too
			fnBuilder.Params(g.buildParams(out, nil, buildSignature)...)
		}
	}

	return fnBuilder
}

func (g *generator) buildParams(params *ast.FieldList, namer Namer, buildSignature bool) []jen.Code {
	var result []jen.Code
	for _, field := range params.List {
		var param *jen.Statement
		if !buildSignature {
			for _, fieldIdent := range field.Names {
				fieldName := fieldIdent.Name
				if fieldName == "" || fieldName == "_" {
					if namer != nil {
						fieldName = namer.New("")
					}
				}
				if param == nil {
					param = jen.Id(fieldName)
				} else {
					param = param.Op(", ").Id(fieldName)
				}
			}
		}
		if param == nil { //anon param (probably return value or _)
			param = &jen.Statement{}
			if namer != nil {
				param.Id(namer.New(""))
			}
		}
		param = g.recursBuildParam(field.Type, param)
		result = append(result, jen.Code(param))
		param = nil
	}
	return result
}

func (g *generator) recursBuildParam(param ast.Expr, root *jen.Statement) *jen.Statement {
	switch exp := param.(type) {
	case *ast.StarExpr:
		g.recursBuildParam(exp.X, root.Op("*"))
	case *ast.Ident:
		if ISScalarType(exp.Name) { //any definitely scalar are set in place others will be resolve after parsing is complete
			return root.Id(exp.Name)
		} else {
			root.Add(g.resolver.Resolve(exp, g.cfg.Package, g.defs[exp]))
		}
	case *ast.Ellipsis:
		g.recursBuildParam(exp.Elt, root.Op("..."))
	case *ast.ArrayType:
		root.Op("[")
		if exp.Len != nil {
			g.recursBuildParam(exp.Len, root)
		}
		root.Op("]")
		g.recursBuildParam(exp.Elt, root)
	case *ast.SelectorExpr:
		if expIdent, ok := exp.X.(*ast.Ident); ok {
			if importSpec := g.localeImport(expIdent.Name); importSpec != nil { //local struct decl
				return root.Qual(importCanon(importSpec.Path.Value), exp.Sel.Name)
			} else {
				log.Println("WARNING: skip *ast.SelectorExpr(recursBuildParam) import of exrp not found", exp.X)
				root.Qual(expIdent.Name, exp.Sel.Name) //it is valid?
			}
		} else {
			log.Println("WARNING: skip *ast.SelectorExpr(recursBuildParam) unsupported format", exp.X)
		}
	case *ast.InterfaceType:
		root.InterfaceFunc(func(group *jen.Group) {
			for _, m := range exp.Methods.List {
				if method, ok := m.Type.(*ast.FuncType); ok {
					for i := range m.Names {
						//shift (remove) func keyword from func abc()
						group.Add(shiftStatement(g.buildFunction(m.Names[i].Name, method.Params, method.Results, nil, false)))
					}
				} else {
					group.Add(g.recursBuildParam(m.Type, &jen.Statement{}))
				}
			}
		})
	case *ast.MapType:
		g.recursBuildParam(exp.Value, root.Map(g.recursBuildParam(exp.Key, &jen.Statement{})))
	case *ast.ChanType:
		if exp.Dir == ast.SEND|ast.RECV {
			g.recursBuildParam(exp.Value, root.Chan())
		} else if exp.Dir&ast.SEND == ast.SEND {
			g.recursBuildParam(exp.Value, root.Chan().Op("<-"))
		} else if exp.Dir&ast.RECV == ast.RECV {
			g.recursBuildParam(exp.Value, root.Op("<-").Chan())
		} else {
			log.Println("WARNING: skip *ast.ChanType(recursBuildParam) unsupported format", exp)
		}
	case *ast.StructType:
		root.StructFunc(func(group *jen.Group) {
			for _, f := range exp.Fields.List {
				fldState := &jen.Statement{}
				for _, fName := range f.Names {
					fldState = fldState.Id(fName.Name).Op(",")
				}
				if len(f.Names) > 0 { //mb len(*fldState)
					fldState = popStatement(fldState) //skip last ,
				}
				statement := g.recursBuildParam(f.Type, fldState)
				if f.Tag != nil {
					statement.Tag(tagToMap(f.Tag))
				}
				group.Add(statement) //todo add other type of tag
			}
		})
	case *ast.FuncType:
		//todo recursive naming resolution
		root.Add(g.buildFunction("", exp.Params, exp.Results, nil, false))
	case *ast.BasicLit:
		root.Id(exp.Value)
	case *ast.BadExpr:
		log.Println("WARNING: bad expression", exp)
	default:
		log.Println(exp.Pos(), exp.End(), exp)
		panic(fmt.Sprintf("unsupported type %s", reflect.ValueOf(exp).String()))
	}
	return root
}

func (g *generator) localeImport(name string) *ast.ImportSpec {
	for _, imprt := range g.imports {
		if imprt.Name == nil {
			if packageName(importCanon(imprt.Path.Value)) == name {
				return imprt
			}
		} else if imprt.Name.Name == name {
			return imprt
		}
	}
	return nil
}

func importCanon(name string) string {
	return strings.ReplaceAll(name, "\"", "")
}

func packageName(packagePath string) string {
	slice := strings.Split(packagePath, "/")
	if len(slice) == 0 {
		return packagePath
	}
	return slice[len(slice)-1]
}

func popStatement(exp *jen.Statement) *jen.Statement {
	proxy := *exp
	if len(proxy) <= 0 {
		return nil
	}
	*exp = proxy[0 : len(proxy)-1]
	return exp
}

func shiftStatement(exp *jen.Statement) *jen.Statement {
	proxy := *exp
	if len(proxy) <= 0 {
		return nil
	}
	*exp = proxy[1:]
	return exp
}

func resetStatement(statement *jen.Statement) *jen.Statement {
	underlying := *statement
	underlying = underlying[0:0]
	*statement = underlying
	return statement
}

func tagToMap(lit *ast.BasicLit) map[string]string {
	if lit == nil {
		return nil
	}
	trimmed := strings.Trim(lit.Value, "`")
	parts := strings.Split(trimmed, ",")
	result := make(map[string]string)
	for _, tags := range parts {
		tagsPart := strings.Split(strings.TrimSpace(tags), ":")
		if len(tagsPart) != 2 {
			log.Println("WARNING: TagToMap tag has unsupported format")
			continue
		}
		//todo other format of tag
		result[tagsPart[0]] = strings.Trim(tagsPart[1], "\"")
	}
	return result
}

func namerFrom(ctx context.Context) Namer {
	if namer, ok := ctx.Value("namer").(Namer); !ok || namer == nil {
		namer = newParameterNamer()
		return namer
	} else {
		return namer
	}
}

func bufferFrom(ctx context.Context, cfg Config) []*wrappedFunctionDeclaration {
	if buffer, ok := ctx.Value("buffer").([]*wrappedFunctionDeclaration); ok && buffer != nil {
		return buffer
	}
	return []*wrappedFunctionDeclaration{}
}

func interfaceWalkFrom(ctx context.Context) bool {
	if _markInterfaceWalk, ok := ctx.Value("_markInterfaceWalk").(bool); ok {
		return _markInterfaceWalk
	}
	return false
}

func deepFrom(ctx context.Context) int {
	if _deep, ok := ctx.Value("_deep").(int); ok {
		return _deep
	}
	return 0
}

func callPrefixFrom(ctx context.Context) []string {
	if _callPrefix, ok := ctx.Value("_callPrefix").([]string); ok {
		return _callPrefix
	}
	return []string{}
}

func withInterfaceWalk(ctx context.Context) context.Context {
	return context.WithValue(ctx, "_markInterfaceWalk", true)
}

func withCallPrefix(ctx context.Context, prefix []string) context.Context {
	return context.WithValue(ctx, "_callPrefix", prefix)
}

func withIncDeep(ctx context.Context) context.Context {
	deep := deepFrom(ctx)
	deep++
	return context.WithValue(ctx, "_deep", deep)
}
