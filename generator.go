package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/dave/jennifer/jen"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"io"
	"log"
	"reflect"
	"strings"
)

var ParserWarning = errors.New("WARNING:")

// generator will work on the selected structure of one file
type generator struct {
	defs     map[*ast.Ident]types.Object
	pkg      *types.Package
	fs       *token.FileSet
	files    []*ast.File
	imports  []*ast.ImportSpec
	namer    *namer
	path     string
	cfg      Config
	inited   bool
	internal bool //seted at runtime in do method
	alice    string
	output   *jen.File
}

func newGenerator(imports []*ast.ImportSpec, files []*ast.File, fs *token.FileSet, path string, cfg Config) *generator {
	return &generator{
		imports: imports,
		cfg:     cfg,
		fs:      fs,
		path:    path,
		files:   files,
		namer:   newNamer(),
	}
}

func (g *generator) init() error {

	/**
	initializing package parsing with the go/type
	*/
	g.defs = make(map[*ast.Ident]types.Object)
	infos := &types.Info{
		Defs: g.defs,
	}

	config := types.Config{Importer: importer.Default(), FakeImportC: true}

	var err error
	g.pkg, err = config.Check(g.path, g.fs, g.files, infos)
	if err != nil {
		log.Println("Warning:", err)
		//return err
	}

	g.inited = true
	return nil
}

func (g *generator) Do(ctx context.Context, target *ast.TypeSpec, targetMethods []ast.Node) error {

	if !g.inited {
		if err := g.init(); err != nil {
			return err
		}
	}

	if target == nil {
		return NotFoundError
	}

	var (
		ok, declared bool
	)

	switch target.Type.(type) {
	case *ast.StructType:
		//nope
	case *ast.InterfaceType:
		//nope
	default:
		return NotFoundError
	}

	var outputBuffer *jen.File
	if outputBuffer, ok = ctx.Value("outputBuff").(*jen.File); !ok || outputBuffer == nil {
		outputBuffer = jen.NewFilePathName(g.cfg.Package, packageName(g.cfg.Package))
		ctx = context.WithValue(ctx, "outputBuff", outputBuffer)
	}

	if g.internal, ok = ctx.Value("_internal").(bool); !ok { //todo cross ref protection
		g.internal = false
		g.alice = g.cfg.Package
	} else {
		if g.alice, ok = ctx.Value("_alice").(string); !ok { //todo cross ref protection
			g.alice = ""
			log.Println("WARNING: internal generator call not supplied with package alice")
		}
	}
	declared = g.internal

	if len(strings.TrimSpace(g.cfg.Comment)) > 0 {
		if g.internal {
			outputBuffer.Line()
			outputBuffer.Comment(fmt.Sprintf("%s from %s", strings.TrimSpace(g.cfg.Comment), g.path))
			outputBuffer.Line()
		} else {
			outputBuffer.PackageComment(strings.TrimSpace(g.cfg.Comment))
			outputBuffer.Line()
		}
	}

	if len(targetMethods) == 0 && !declared {
		//outputBuffer.Comment("first")
		if _, ok = target.Type.(*ast.InterfaceType); ok {
			outputBuffer.Var().Id(g.cfg.Variable).Id(target.Name.Name).Line()
		} else {
			outputBuffer.Var().Id(g.cfg.Variable).Op("*").Id(target.Name.Name).Line()
		}
		declared = true
	}

	for i := range targetMethods {
		switch method := targetMethods[i].(type) {
		case *ast.FuncDecl:
			if method.Name.IsExported() {
				if !declared {
					//outputBuffer.Comment("second")
					switch mType := method.Recv.List[0].Type.(type) {
					case *ast.StarExpr: //mb [T]
						outputBuffer.Var().Id(g.cfg.Variable).Op("*").Id(mType.X.(*ast.Ident).Name).Line()
					case *ast.Ident:
						outputBuffer.Var().Id(g.cfg.Variable).Id(mType.Name).Line()
					}
					declared = true
				}
				g.wrapFunction(outputBuffer, method.Name.Name, method.Type.Params, method.Type.Results, method.Doc.Text())
				outputBuffer.Line()
			}
		default:
			log.Println("ubnormal value in targetMethods", reflect.ValueOf(targetMethods[i]).String())
		}
	}

	switch structure := target.Type.(type) {
	case *ast.StructType:
		for _, field := range structure.Fields.List {
			if g.isIgnored(field.Tag) {
				continue
			}
			if err := g.digField(ctx, outputBuffer, field, field.Type); err != nil {
				if errors.Is(err, ParserWarning) {
					log.Println(err)
					continue
				} else {
					return err
				}
			}
		}
	case *ast.InterfaceType:
		for _, field := range structure.Methods.List {
			if g.isIgnored(field.Tag) {
				continue
			}
			if err := g.digField(ctx, outputBuffer, field, field.Type); err != nil {
				if errors.Is(err, ParserWarning) {
					log.Println(err)
					continue
				} else {
					return err
				}
			}
		}
	default:
		panic("must not happened")
	}

	g.output = outputBuffer //todo remove

	return nil
}

func (g *generator) WriteTo(writer io.Writer) error {
	if g.output != nil {
		return g.output.Render(writer)
	}
	return nil
}

func (g *generator) digField(ctx context.Context, outputBuffer *jen.File, field *ast.Field, inner ast.Expr) error {
	switch fieldTyped := inner.(type) {
	case *ast.FuncType:
		if len(field.Names) > 0 && field.Names[0].IsExported() {
			g.wrapFunction(outputBuffer, field.Names[0].Name, fieldTyped.Params, fieldTyped.Results, field.Doc.Text())
			outputBuffer.Line()
		}
	case *ast.StarExpr:
		g.digField(ctx, outputBuffer, field, fieldTyped.X) //keep *
	case *ast.SelectorExpr:
		if len(field.Names) == 0 {
			if err := g.digExternalDecl(ctx, fieldTyped); err != nil {
				return err
			}
		}
	case *ast.Ident:
		if len(field.Names) == 0 {
			if fieldTyped.Obj != nil && fieldTyped.Obj.Kind == ast.Typ {
				if _, ok := fieldTyped.Obj.Decl.(*ast.TypeSpec).Type.(*ast.StructType); ok {
					err := g.parsePackage(ctx, g.cfg.Package, fieldTyped.Name, fmt.Sprintf("<%s>", field.Type))
					if err != nil {
						return err
					}
				}
			}
		}
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
		err := g.parsePackage(ctx, importCanon(importSpec.Path.Value), str.Sel.Name, fmt.Sprintf("<%s>", str))
		if err != nil {
			return err
		}
		return nil
	} else {
		return fmt.Errorf("%w skip *ast.SelectorExpr(do) unsupported format %v", ParserWarning, str.X)
	}
}

func (g *generator) parsePackage(ctx context.Context, pkg, structure, comment string) error {
	config := g.cfg
	config.Package = importCanon(pkg)
	config.Structure = structure
	config.Comment = comment
	if config.Deep > 0 {
		config.Deep--
		if config.Deep == 0 {
			return fmt.Errorf("%s:%s skipped as too deep inspect", pkg, structure)
		}
	}
	ctx = context.WithValue(ctx, "_internal", true)
	ctx = context.WithValue(ctx, "_alice", config.Package)
	err := parsePackage(ctx, config)
	if err != nil {
		return err
	}
	return nil
}

func (g *generator) digInternalDecl(ctx context.Context, structure, comment string) error {
	ctx = context.WithValue(ctx, "_internal", true)
	ctx = context.WithValue(ctx, "_alice", g.cfg.Package)

	//sf := newStructFinder(structure, g.path)
	//ast.Inspect(g., sf.find)

	g.Do(ctx, &ast.TypeSpec{}, []ast.Node{})

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

func (g *generator) wrapFunction(buffer *jen.File, name string, in, out *ast.FieldList, comment string) {
	if len(comment) > 0 {
		buffer.Comment(comment)
	}
	g.namer.Reset()
	fnBuilder := jen.Func().Add(g.buildFunctionSignature(name, in, out, true))
	underFn := jen.Id(g.cfg.Variable).Dot(name).CallFunc(func(group *jen.Group) {
		for _, field := range in.List {
			for _, fieldIdent := range g.namer.Values() {
				group.Add(jen.Id(fieldIdent))
			}
			g.namer.Reset()
			for _, fieldIdent := range field.Names {
				if _, ok := field.Type.(*ast.Ellipsis); ok {
					group.Add(jen.Id(fieldIdent.Name).Op("..."))
				} else {
					group.Add(jen.Id(fieldIdent.Name))
				}
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
	buffer.Add(fnBuilder)
}

func (g *generator) buildFunctionSignature(name string, in, out *ast.FieldList, generateNameForAnon bool) *jen.Statement {
	fnBuilder := jen.Id(name)
	if in.NumFields() > 0 {
		fnBuilder.Params(g.buildParams(true && generateNameForAnon, in)...)
	} else {
		fnBuilder.Op("()")
	}
	outFieldsCnt := out.NumFields()
	if out != nil && out.NumFields() > 0 {
		if outFieldsCnt == 1 {
			fnBuilder.List(g.buildParams(false && generateNameForAnon, out)...)
		} else {
			fnBuilder.Params(g.buildParams(false && generateNameForAnon, out)...)
		}
	}

	return fnBuilder
}

func (g *generator) buildParams(generateNameForAnon bool, params *ast.FieldList) []jen.Code {
	var result []jen.Code
	for _, field := range params.List {
		var param *jen.Statement
		for _, fieldIdent := range field.Names {
			if param == nil {
				param = jen.Id(fieldIdent.Name)
			} else {
				param = param.Op(", ").Id(fieldIdent.Name)
			}
		}
		if param == nil { //anon param (probably return value)
			param = &jen.Statement{}
			if generateNameForAnon {
				param.Id(g.namer.New(""))
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
		if exp.Obj == nil { //it probably scalar //ISScalarType(exp.Name) { //todo check Obj == nil
			return root.Id(exp.Name)
		} else if true { //local struct decl
			return root.Qual(g.alice, exp.Name)
		} else { //for generics
			log.Println("WARNING: skip *ast.SelectorExpr(recursBuildParam) unsupported format")
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
					group.Add(g.buildFunctionSignature(m.Names[0].Name, method.Params, method.Results, false))
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
					fldState = exprUndo(fldState) //skip last ,
				}
				statement := g.recursBuildParam(f.Type, fldState)
				if f.Tag != nil {
					statement.Tag(tagToMap(f.Tag))
				}
				group.Add(statement) //todo add other type of tag
			}
		})
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

func exprUndo(exp *jen.Statement) *jen.Statement {
	proxy := *exp
	if len(proxy) <= 0 {
		return nil
	}
	proxy = proxy[0 : len(proxy)-1]
	return &proxy
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
