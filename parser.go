package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/dave/jennifer/jen"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
)

var (
	ProcessedError = errors.New("target processed")
	NotFoundError  = errors.New("not found")
)

type vType uint

const (
	Auto vType = iota
	Real
	Ref
)

type Config struct {
	Deep         int
	Package      string
	Target       string
	Variable     string
	Comment      string
	Write        bool
	Suffix, Path string
}

type loaderRecord struct {
	*types.Package
	packageDefs
	files   map[string]*ast.File
	fileSet *token.FileSet
	targets []string
	path    string
	inited  bool
	sync.Mutex
}
type loaderRecords map[string]*loaderRecord
type pendingParserReq struct {
	Config
	context.Context
}

type packageDefs map[*ast.Ident]types.Object

var (
	pending    []*pendingParserReq
	pendingMux               = sync.Mutex{}
	records    loaderRecords = make(loaderRecords)
)

var mut sync.Mutex

// this is not realy good aproach to service locator / di
// because of context behavior and go inteface nil behavior
// and general non-obviousness internal checks
// but we have what we have
func SetupCtx(
	ctx context.Context,
	writer io.Writer,
	buffer *jen.File,
	namer Namer,
	checker Checker,
	blackList []string,
	params ...any,
) context.Context {
	var (
		asDefault, ok bool
	)
	if len(params) >= 1 {
		//first param is asDefault ie reverse logic
		if asDefault, ok = params[0].(bool); !ok {
			asDefault = false
		}
	}
	//log.Println(ctx, asDefault, ok)
	//todo nil check
	if test, ok := ctx.Value("writer").(io.Writer); !asDefault || (!ok || test == nil) {
		//log.Println("setup writer")
		ctx = context.WithValue(ctx, "writer", writer)
	}
	//warning: if test is any and *jen.File is nil then {inteface{}}nil != nil
	if test, ok := ctx.Value("buffer").(*jen.File); !asDefault || (!ok || test == nil) {
		//log.Println("setup buffer")
		ctx = context.WithValue(ctx, "buffer", buffer)
	} else {
		//log.Println(test == nil, ok, asDefault)
	}
	if test, ok := ctx.Value("namer").(Namer); !asDefault || (!ok || test == nil) {
		//log.Println("setup namer")
		ctx = context.WithValue(ctx, "namer", namer)
	}
	if test, ok := ctx.Value("checker").(Checker); !asDefault || (!ok || test == nil) {
		//log.Println("setup checker")
		ctx = context.WithValue(ctx, "checker", checker)
	}
	if test, ok := ctx.Value("blackList").([]string); !asDefault || (!ok || test == nil) {
		//log.Println("setup checker")
		ctx = context.WithValue(ctx, "blackList", blackList)
	}

	return ctx
}

// ParsePackage launch the generation
func ParsePackage(ctx context.Context, cfg Config) error {

	var (
		writer         io.Writer
		totalGenerated []*wrappedFunctionDeclaration
		err            error
	)
	mut.Lock()
	records = make(loaderRecords)
	mut.Unlock()
	pendingMux.Lock()
	pending = make([]*pendingParserReq, 0)
	pendingMux.Unlock()

	cfg.Package = strings.TrimSpace(cfg.Package)
	cfg.Target = strings.TrimSpace(cfg.Target)
	cfg.Variable = strings.TrimSpace(cfg.Variable)
	cfg.Suffix = strings.TrimSpace(cfg.Suffix)
	cfg.Path = strings.TrimSpace(cfg.Path)
	cfg.Comment = strings.TrimSpace(cfg.Comment)

	if len(cfg.Package) == 0 {
		return errors.New("no directory submitted")
	}

	if len(cfg.Target) == 0 {
		return errors.New("no target submitted")
	}

	if len(cfg.Variable) == 0 {
		cfg.Variable = "Instance"
	}

	if len(cfg.Suffix) == 0 {
		cfg.Suffix = "_singleton.go"
	}

	if ctx.Value("writer") != nil || len(cfg.Path) > 0 || cfg.Suffix != "_singleton.go" {
		cfg.Write = true
	}

	// get the path of the package
	if strings.TrimSpace(os.Getenv("GOPATH")) == "" {
		caution("WARNING: OS ENV GOPATH NOT SET!")
	}
	if strings.TrimSpace(os.Getenv("GOROOT")) == "" {
		caution("WARNING: OS ENV GOROOT NOT SET!")
	}

	info("parse package", cfg.Package, cfg.Target)

	buffer := jen.NewFilePathName(cfg.Package, packageName(cfg.Package))
	//loader := recursiveLoaderBuilder(buffer, cfg)
	loader := linearLoaderBuilder(buffer, cfg)

	varDecl := newVariableDeclFromConfig(cfg)
	cfg.Variable = clearVarDeclaration(cfg.Variable)

	ctx = SetupCtx(ctx,
		nil,
		buffer,
		newParameterNamer(),
		newUniqueChecker(nil),
		[]string{"_test.go", cfg.Suffix}, //todo merge
		true,                             //it sets as default
	)

	if len(cfg.Comment) > 0 {
		buffer.PackageComment(cfg.Comment)
		buffer.Line()
	}

	//variable placeholder will be updated later
	buffer.Var().Add(varDecl.Declare())
	ctx = withPending(ctx, []Delayed{varDecl})
	ctx = withResolver(ctx, varDecl.rootResolver)

	cfg.Comment = fmt.Sprintf("<%s>", cfg.Target)
	resultChanel := make(chan struct {
		context.Context
		Decl []*wrappedFunctionDeclaration
		Config
		error
	}, 10)

	go generateRoutine(ctx, loader, resultChanel, cfg)

	result := <-resultChanel

	if result.error != nil {
		return result.error
	}
	ctx = result.Context

	totalGenerated = append(totalGenerated, result.Decl...)

	//generate dep tree packages begin

	untilEnd := cfg.Deep == 0
	undergoingTask := 0
	originalOrder := make([]*pendingParserReq, 0)
	generatedParts := make(map[Config][]*wrappedFunctionDeclaration, 0)
	for len(pending) > 0 /*&& (untilEnd || turnsLeft > 0)*/ {

		pendingMux.Lock()
		newTasks := make([]*pendingParserReq, len(pending))
		copy(newTasks, pending)
		pending = pending[0:0]
		pendingMux.Unlock()

		for _, task := range newTasks {
			if untilEnd || task.Deep > 0 {
				go generateRoutine(task.Context, loader, resultChanel, task.Config)
				undergoingTask++
			}
		}
		originalOrder = append(originalOrder, newTasks...) //keep original order

	wait:
		for {
			select {
			case result = <-resultChanel:
				undergoingTask--
				if result.error != nil {
					if !errors.Is(result.error, ProcessedError) {
						info(result.error)
					}
				}
				//result.Context not needed
				generatedParts[result.Config] = result.Decl
			default:
				//len is not atomic but undergoingTask protects against problems
				if len(pending) > 0 || undergoingTask == 0 {
					break wait
				}
			}
		}
	}

	close(resultChanel)

	//restore original order
	for _, task := range originalOrder {
		totalGenerated = append(totalGenerated, generatedParts[task.Config]...)
	}
	generatedParts, originalOrder = nil, nil

	checker := chekerFrom(ctx).NewChecker(totalGenerated)
	glue(buffer, checker.Valid(), cfg)

	varDecl.CompleteResolve() //resolve all pending decl

	writer, done, fail, err := setupOutput(ctx, cfg)

	if err = buffer.Render(writer); err != nil {
		fail()
		log.Println(err)
	} else {
		done()
	}

	checkerErrors := checker.Invalid() //if it not in use it usually will be empty
	if len(checkerErrors) > 0 {
		fmt.Println("")
		info(fmt.Sprintf("Followed checker errors appears while parsing package %s (probably duplication of declaration) it was dropped from output", cfg.Package))
	}
	for _, fn := range checkerErrors {
		caution(fn.Signature)
	}
	if len(checkerErrors) > 0 {
		fmt.Println("")
	}
	return nil
}

func linearLoaderBuilder(buffer *jen.File, cfg Config) loaderCallback {

	var linearLoader loaderCallback
	linearLoader = func(ctx context.Context, cfg Config) error {
		pendingMux.Lock()
		pending = append(pending, &pendingParserReq{
			Config:  cfg,
			Context: ctx,
		})
		pendingMux.Unlock()
		return nil
	}

	return linearLoader
}

func glue(output *jen.File, content []*wrappedFunctionDeclaration, cfg Config) {
	var (
		pkg, target string
	)

	for _, fn := range content {
		if pkg != fn.Package && target != fn.Target {
			output.Line().Comment(fmt.Sprintf("%s from %s", strings.TrimSpace(fn.Comment), fn.Package)).Line()
			pkg, target = fn.Package, fn.Target
		}
		for _, statement := range fn.Content {
			output.Add(statement)
			output.Line()
		}
	}
}

func setupOutput(ctx context.Context, cfg Config) (writer io.Writer, done, fail func(), err error) {
	var (
		resetFile *os.File
		ok        bool
	)
	done = func() {}
	fail = func() {}
	writer = os.Stdout
	if writer, ok = ctx.Value("writer").(io.Writer); writer == nil || !ok {
		if !cfg.Write {
			writer = os.Stdout
			ctx = context.WithValue(ctx, "writer", writer)
		} else {
			var resetFilePath string
			if cfg.Path == "" { //todo path validation
				p, err := build.Default.Import(cfg.Package, ".", build.FindOnly)
				if err != nil {
					panic(err)
				}
				//resetFilePath = p.Dir + "/" + packageName(importCanon(cfg.Package)) + cfg.Suffix
				//todo encoding of filepath
				resetFilePath = p.Dir + "/" + strings.ToLower(cfg.Target[0:1]) + cfg.Target[1:] + cfg.Suffix
			} else {
				resetFilePath = cfg.Path
			}
			// delete if needed
			_ = os.Remove(resetFilePath)
			// writeType to a file
			resetFile, err = os.OpenFile(resetFilePath, os.O_CREATE|os.O_RDWR, 0600)
			if err != nil {
				return writer, done, fail, err
			}
			writer = resetFile
			ctx = context.WithValue(ctx, "writer", writer)
			done = func() {
				resetFile.Close()
			}
			fail = func() {
				resetFile.Close()
				_ = os.Remove(resetFile.Name())
			}
		}
	}
	return writer, done, fail, err
}

func collectFiles(pkg string, blacklist []string) (path string, files map[string]*ast.File, fileSet *token.FileSet, err error) {
	var (
		packages map[string]*ast.Package
		p        *build.Package
	)

	p, err = build.Default.Import(pkg, ".", build.FindOnly)
	if err != nil {
		return "", nil, nil, err
	}

	path = p.Dir

	fileSet = token.NewFileSet()
	packages, err = parser.ParseDir(fileSet, path, nil, 0)

	if err != nil {
		return "", nil, nil, err
	}

	files = make(map[string]*ast.File)
	for pPath := range packages {
		for j := range packages[pPath].Files {
			if !isBlackListed(j, blacklist) {
				files[j] = packages[pPath].Files[j]
			} else {
				//log.Println(j, "ignored")
			}
		}
	}

	return path, files, fileSet, nil
}

func generateRoutine(ctx context.Context, loader loaderCallback, output chan<- struct {
	context.Context
	Decl   []*wrappedFunctionDeclaration
	Config //in order to identify part
	error
}, cfg Config) {
	var (
		p   *loaderRecord
		ok  bool
		err error
	)

	mut.Lock()
	if p, ok = records[cfg.Package]; !ok {
		p = &loaderRecord{
			files:   nil,
			targets: []string{},
			path:    "",
			inited:  false,
			Mutex:   sync.Mutex{},
		}
		records[cfg.Package] = p
	}
	mut.Unlock()

	p.Lock()
	if !p.inited {
		records[cfg.Package].path, records[cfg.Package].files, records[cfg.Package].fileSet, err = collectFiles(cfg.Package, blackListFrom(ctx))
		if err != nil {
			output <- struct {
				context.Context
				Decl []*wrappedFunctionDeclaration
				Config
				error
			}{Context: ctx, Decl: []*wrappedFunctionDeclaration{}, Config: cfg, error: err}
			return
		}
		records[cfg.Package].Package, records[cfg.Package].packageDefs, err = initPackage(records[cfg.Package].path, records[cfg.Package].files, records[cfg.Package].fileSet)
		p.inited = true
	}
	for _, target := range records[cfg.Package].targets {
		if target == cfg.Target {
			p.Unlock()
			output <- struct {
				context.Context
				Decl []*wrappedFunctionDeclaration
				Config
				error
			}{Context: ctx, Decl: []*wrappedFunctionDeclaration{}, Config: cfg, error: ProcessedError}
			return
		}
	}
	records[cfg.Package].targets = append(records[cfg.Package].targets, cfg.Target)
	p.Unlock()

	indexes := make([]string, 0, len(records[cfg.Package].files))
	for path := range records[cfg.Package].files {
		indexes = append(indexes, path)
	}
	sort.Strings(indexes)
	generatedTotal := make([]*wrappedFunctionDeclaration, 0)
	targetFound := false
	for _, file := range indexes {
		sf := newStructFinder(cfg.Target, cfg.Package)
		ast.Inspect(records[cfg.Package].files[file], sf.Find)
		if sf.Structure() != nil || len(sf.Methods()) > 0 {
			gen := newGenerator(sf.Imports(), cfg, loader, records[cfg.Package].packageDefs, records[cfg.Package].path)
			if ctx, err = gen.Do(ctx, sf.Structure(), sf.Methods()); err != nil {
				info(err)
				continue
			}
			generatedTotal = append(generatedTotal, gen.Result()...)
			if sf.Structure() != nil {
				targetFound = true
			}
		}
	}

	if targetFound {
		output <- struct {
			context.Context
			Decl []*wrappedFunctionDeclaration
			Config
			error
		}{Context: ctx, Decl: generatedTotal, Config: cfg, error: nil}
	} else {
		output <- struct {
			context.Context
			Decl []*wrappedFunctionDeclaration
			Config
			error
		}{Context: ctx, Decl: []*wrappedFunctionDeclaration{}, Config: cfg, error: fmt.Errorf("%s: %s %w", cfg.Package, cfg.Target, NotFoundError)}
	}

	return
}

func initPackage(path string, files map[string]*ast.File, fs *token.FileSet) (*types.Package, packageDefs, error) {

	/**
	initializing package parsing with the go/type
	*/

	defs := make(map[*ast.Ident]types.Object)
	infos := &types.Info{
		Defs: defs,
	}

	var errorsCnt int
	config := types.Config{
		FakeImportC: true,
		Error: func(err error) {
			//log.Println(err)
		},
		Importer: importer.Default(),
	}

	if errorsCnt > 0 {
		info("Warning: During parsing via config.Check number of errors appeared", errorsCnt)
	}

	var err error

	pkg, err := config.Check(path, fs, mapToSlice(files), infos)

	if err != nil {
		log.Println("Warning:", err)
		//return err
	}

	return pkg, packageDefs(defs), nil
}

// todo refactor
func clearVarDeclaration(string2 string) string {
	string2 = strings.Replace(strings.Replace(string2, "*", "", 1), "&", "", 1)
	start, end := strings.Index(string2, "["), strings.Index(string2, "]")
	if start < 0 || end < 0 {
		return string2
	}
	return string2[0:start]
}

func isBlackListed(f string, blacklist []string) bool {
	for _, rec := range blacklist {
		if strings.Contains(f, rec) {
			return true
		}
	}
	return false
}

func blackListFrom(ctx context.Context) []string {
	if checker, ok := ctx.Value("blackList").([]string); ok && checker != nil {
		return checker
	}
	return []string{}
}

func mapToSlice[Key comparable, Value any](from map[Key]Value) []Value {
	to := make([]Value, 0, len(from))
	for i := range from {
		to = append(to, from[i])
	}
	return to
}

func pendingStatementFrom(ctx context.Context) []Delayed {
	if _statement, ok := ctx.Value("_delayed").([]Delayed); ok {
		return _statement
	}
	return nil
}

func resolverFrom(ctx context.Context) Resolver {
	if _statement, ok := ctx.Value("_resolver").(Resolver); ok {
		return _statement
	}
	return nil
}

func withPending(ctx context.Context, delayed []Delayed) context.Context {
	return context.WithValue(ctx, "_delayed", delayed)
}

func withResolver(ctx context.Context, resolver Resolver) context.Context {
	return context.WithValue(ctx, "_resolver", resolver)
}

func chekerFrom(ctx context.Context) Checker {
	if checker, ok := ctx.Value("checker").(Checker); ok && checker != nil {
		return checker
	}
	return newUniqueChecker(nil) //must not be happened
}

func info(text ...any) {
	log.Println(text...)
}

func caution(text any) {
	log.Println(color(Yellow, fmt.Sprintf("%s", text)))
}

func critical(text any) {
	log.Println(color(Red, fmt.Sprintf("%s", text)))
}
