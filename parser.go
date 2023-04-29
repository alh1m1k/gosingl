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
)

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
	pending    []pendingParserReq
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
		ok             bool
		resetFile      *os.File
		totalGenerated []*wrappedFunctionDeclaration
		err            error
	)
	mut.Lock()
	records = make(loaderRecords)
	mut.Unlock()
	pendingMux.Lock()
	pending = make([]pendingParserReq, 0)
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
		info("WARNING: OS ENV GOPATH NOT SET!")
	}
	if strings.TrimSpace(os.Getenv("GOROOT")) == "" {
		info("WARNING: OS ENV GOROOT NOT SET!")
	}

	buffer := jen.NewFilePathName(cfg.Package, packageName(cfg.Package))
	//loader := recursiveLoaderBuilder(buffer, cfg)
	loader := linearLoaderBuilder(buffer, cfg)

	ctx = SetupCtx(ctx,
		nil,
		buffer,
		newParameterNamer(),
		nil,
		[]string{"_test.go", cfg.Suffix}, //todo merge
		true,                             //it sets as default
	)

	if len(cfg.Comment) > 0 {
		buffer.PackageComment(strings.TrimSpace(cfg.Comment))
		buffer.Line()
	}

	//variable placeholder will be updated later
	varDeclPlaceholder := buffer.Var()
	varDeclPlaceholder.Id(cfg.Variable).Op("*").Id(cfg.Target).Line() //default
	ctx = withPendingStatement(ctx, "var", varDeclPlaceholder)

	cfg.Comment = fmt.Sprintf("<%s>", cfg.Target)
	resultChanel := make(chan struct {
		context.Context
		Decl []*wrappedFunctionDeclaration
		error
	}, 10)

	go generateRoutine(ctx, loader, resultChanel, cfg)

	result := <-resultChanel

	if result.error != nil {
		//log.Println("die")
		return result.error
	}

	if pendingStatementFrom(ctx, "var") != nil {
		critical("WARNING: unable to determinate target variable type, left as default (Ref)")
	}
	totalGenerated = append(totalGenerated, result.Decl...)

	//untilEnd := cfg.Deep == 0
	//turnsLeft := cfg.Deep

	undergoingTask := 0
	for len(pending) > 0 /*&& (untilEnd || turnsLeft > 0)*/ {

		pendingMux.Lock()
		newTasks := make([]pendingParserReq, len(pending))
		copy(newTasks, pending)
		pending = pending[0:0]
		pendingMux.Unlock()

		for _, task := range newTasks {
			//log.Println("run routine", task.Package, task.Target)
			go generateRoutine(task.Context, loader, resultChanel, task.Config)
			undergoingTask++
		}

	wait:
		for {
			select {
			case result = <-resultChanel:
				undergoingTask--
				if result.error != nil {
					if !errors.Is(err, ProcessedError) {
						info(err)
					}
				}
				totalGenerated = append(totalGenerated, result.Decl...)
			default:
				//todo is len atomic?
				if len(pending) > 0 || undergoingTask == 0 {
					break wait
				}
			}
		}

		//turnsLeft--
	}

	checker := chekerFrom(ctx, totalGenerated)
	glue(buffer, checker.Valid())

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
				resetFilePath = p.Dir + "/" + strings.ToLower(cfg.Target[0:1]) + cfg.Target[1:] + cfg.Suffix
			} else {
				resetFilePath = cfg.Path
			}
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

	if err = buffer.Render(writer); err != nil {
		log.Println(err)
	}

	checkerErrors := checker.Invalid() //if it not in use it usually will be empty
	if len(checkerErrors) > 0 {
		info("Followed checker errors appears while parsing package (probably duplication of declaration) it was dropped from output")
	}
	for _, fn := range checkerErrors {
		critical(fn.Signature)
	}
	return nil
}

func linearLoaderBuilder(buffer *jen.File, cfg Config) func(ctx context.Context, pkg, structure, comment string) error {

	var linearLoader loaderCallback
	linearLoader = func(ctx context.Context, pkg, structure, comment string) error {
		//log.Println(pkg, structure)
		config := Config{
			Deep:     0,
			Package:  importCanon(pkg),
			Variable: cfg.Variable,
			Target:   structure,
			Comment:  comment,
		}
		pendingMux.Lock()
		pending = append(pending, pendingParserReq{
			Config:  config,
			Context: ctx,
		})
		pendingMux.Unlock()
		return nil
	}

	return linearLoader
}

func recursiveLoaderBuilder(buffer *jen.File, cfg Config) func(ctx context.Context, pkg, structure, comment string) error {
	//todo deep counter respect
	var recursiveLoader loaderCallback
	recursiveLoader = func(ctx context.Context, pkg, structure, comment string) error {
		config := Config{
			Deep:     0,
			Package:  importCanon(pkg),
			Variable: cfg.Variable,
			Target:   structure,
			Comment:  comment,
		}
		_, _, err := generate(ctx, recursiveLoader, config) //break context chain
		return err
	}

	return recursiveLoader
}

func glue(output *jen.File, content []*wrappedFunctionDeclaration) {
	var (
		pkg, target string
	)
	sort.SliceStable(content, func(i, j int) bool {
		return content[i].Package+"_"+content[i].Target < content[j].Package+"_"+content[j].Target
	})
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

func generate(ctx context.Context, loader loaderCallback, cfg Config) (context.Context, []*wrappedFunctionDeclaration, error) {
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
			return ctx, []*wrappedFunctionDeclaration{}, err
		}
		//initPackage is @deprecated and will be removed in future
		records[cfg.Package].Package, records[cfg.Package].packageDefs, err = initPackage(records[cfg.Package].path, records[cfg.Package].files, records[cfg.Package].fileSet)
		p.inited = true
	}
	for _, target := range records[cfg.Package].targets {
		if target == cfg.Target {
			p.Unlock()
			return ctx, []*wrappedFunctionDeclaration{}, ProcessedError
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
	for _, file := range indexes {
		sf := newStructFinder(cfg.Target, cfg.Package)
		ast.Inspect(records[cfg.Package].files[file], sf.find)
		if sf.structure() != nil || len(sf.methods()) > 0 {
			gen := newGenerator(sf.imports(), cfg, loader, records[cfg.Package].packageDefs, records[cfg.Package].path)
			if ctx, err = gen.Do(ctx, sf.structure(), sf.methods()); err != nil {
				info(err)
				continue
			}
			generatedTotal = append(generatedTotal, gen.Result()...)
		}
	}

	return ctx, generatedTotal, nil
}

func generateRoutine(ctx context.Context, loader loaderCallback, output chan<- struct {
	context.Context
	Decl []*wrappedFunctionDeclaration
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
				error
			}{Context: ctx, Decl: []*wrappedFunctionDeclaration{}, error: err}
		}
		//initPackage is @deprecated and will be removed in future
		records[cfg.Package].Package, records[cfg.Package].packageDefs, err = initPackage(records[cfg.Package].path, records[cfg.Package].files, records[cfg.Package].fileSet)
		p.inited = true
	}
	for _, target := range records[cfg.Package].targets {
		if target == cfg.Target {
			p.Unlock()
			output <- struct {
				context.Context
				Decl []*wrappedFunctionDeclaration
				error
			}{Context: ctx, Decl: []*wrappedFunctionDeclaration{}, error: ProcessedError}
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
	for _, file := range indexes {
		sf := newStructFinder(cfg.Target, cfg.Package)
		ast.Inspect(records[cfg.Package].files[file], sf.find)
		if sf.structure() != nil || len(sf.methods()) > 0 {
			gen := newGenerator(sf.imports(), cfg, loader, records[cfg.Package].packageDefs, records[cfg.Package].path)
			if ctx, err = gen.Do(ctx, sf.structure(), sf.methods()); err != nil {
				info(err)
				continue
			}
			generatedTotal = append(generatedTotal, gen.Result()...)
		}
	}

	output <- struct {
		context.Context
		Decl []*wrappedFunctionDeclaration
		error
	}{Context: ctx, Decl: generatedTotal, error: nil}
}

func initPackage(path string, files map[string]*ast.File, fs *token.FileSet) (*types.Package, packageDefs, error) {

	/**
	initializing package parsing with the go/type
	*/

	defs := make(map[*ast.Ident]types.Object)
	infos := &types.Info{
		Defs: defs,
	}

	errorsCnt := 0
	config := types.Config{
		FakeImportC: true,
		Error: func(err error) {
			errorsCnt++
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

func pendingStatementFrom(ctx context.Context, name string) *jen.Statement {
	if _statement, ok := ctx.Value(fmt.Sprintf("_pending:%s", name)).(*jen.Statement); ok {
		return _statement
	}
	return nil
}

func withPendingStatement(ctx context.Context, name string, statement *jen.Statement) context.Context {
	return context.WithValue(ctx, fmt.Sprintf("_pending:%s", name), statement)
}

func chekerFrom(ctx context.Context, decl []*wrappedFunctionDeclaration) Checker {
	if checker, ok := ctx.Value("checker").(Checker); ok && checker != nil {
		return checker
	}
	return newUniqueChecker2(decl) //must not be happened
}

func info(text ...any) {
	log.Println(text...)
}

func critical(text any) {
	log.Println(color(Yellow, fmt.Sprintf("%s", text)))
}

func stop(text any) {
	log.Println(color(Red, fmt.Sprintf("%s", text)))
}
