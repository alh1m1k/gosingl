package main

import (
	"context"
	"errors"
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
	"strings"
	"sync"
)

var (
	ProcessedError = errors.New("target processed")
)

type loaderRecord struct {
	*types.Package
	packageDefs
	files   []*ast.File
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

func loader(ctx context.Context, cfg Config) {
	pending = append(pending, pendingParserReq{
		Config:  cfg,
		Context: ctx,
	})
}

var mut sync.Mutex

// this is not realy good aproach to service locator / di
// because of context behavior and go inteface nil behavior
// and general non-obviousness of internal checks
// but we have what we have
func SetupCtx(
	ctx context.Context,
	writer io.Writer,
	buffer *jen.File,
	namer Namer,
	checker Checker,
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

	return ctx
}

// ParsePackage launchs the generation
func ParsePackage(ctx context.Context, cfg Config) error {

	var (
		writer    io.Writer
		ok        bool
		resetFile *os.File
		err       error
	)

	if len(strings.TrimSpace(cfg.Package)) <= 0 {
		return errors.New("no directory submitted")
	}

	if len(strings.TrimSpace(cfg.Variable)) <= 0 {
		//return errors.New("instance empty submitted")
	}

	if ctx.Value("writer") != nil {
		cfg.Write = true
	}

	// get the path of the package
	if strings.TrimSpace(os.Getenv("GOPATH")) == "" {
		log.Println("WARNING: OS ENV GOPATH NOT SET!")
	}
	if strings.TrimSpace(os.Getenv("GOROOT")) == "" {
		log.Println("WARNING: OS ENV GOROOT NOT SET!")
	}

	buffer := jen.NewFilePathName(cfg.Package, packageName(cfg.Package))
	//loader := recursiveLoaderBuilder(buffer, cfg)
	loader := linearLoaderBuilder(buffer, cfg)

	/*	ctx = SetupCtx(context.Background(), //as reference
		nil,
		nil,
		nil,
		nil,
	)*/

	checker := newUniqueChecker()
	ctx = SetupCtx(ctx,
		nil,
		buffer,
		newParameterNamer(),
		checker,
		true, //as default
	)

	if err = generate(ctx, loader, cfg); err != nil {
		return err
	}

	untilEnd := cfg.Deep == 0
	turnsLeft := cfg.Deep
	for len(pending) > 0 && (untilEnd || turnsLeft > 0) {

		pendingMux.Lock()
		newTasks := make([]pendingParserReq, len(pending))
		copy(newTasks, pending)
		pending = pending[0:0]
		pendingMux.Unlock()

		for _, task := range newTasks {
			//log.Println(task.Package, task.Structure)
			nCtx := context.WithValue(task.Context, "_internal", true)
			nCtx = context.WithValue(nCtx, "_alice", task.Package)
			if err = generate(nCtx, loader, task.Config); err != nil {
				if !errors.Is(err, ProcessedError) {
					log.Println(err)
				}
				continue
			}
		}

		turnsLeft--
	}

	if writer, ok = ctx.Value("writer").(io.Writer); writer == nil || !ok {
		if !cfg.Write {
			writer = os.Stdout

			ctx = context.WithValue(ctx, "writer", writer)
		} else {
			p, err := build.Default.Import(cfg.Package, ".", build.FindOnly)
			if err != nil {
				panic(err)
			}
			resetFilePath := p.Dir + "/" + packageName(importCanon(cfg.Package)) + "_singleton.go"
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

	checkerErrors := checker.Errors() //if it not in use it usually will be empty
	if len(checkerErrors) > 0 {
		log.Println("Followed checker errors appears while parsing package (probably duplication of declaration) it was skipped")
	}
	for _, err := range checkerErrors {
		log.Println(err)
	}
	return nil
}

func linearLoaderBuilder(buffer *jen.File, cfg Config) func(ctx context.Context, pkg, structure, comment string) error {

	var linearLoader loaderCallback
	linearLoader = func(ctx context.Context, pkg, structure, comment string) error {
		config := Config{
			Deep:      0,
			Package:   importCanon(pkg),
			Variable:  cfg.Variable,
			Structure: structure,
			Comment:   comment,
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
			Deep:      0,
			Package:   importCanon(pkg),
			Variable:  cfg.Variable,
			Structure: structure,
			Comment:   comment,
		}
		return generate(ctx, recursiveLoader, config)
	}

	return recursiveLoader
}

func collectFiles(pkg string) (path string, files []*ast.File, fileSet *token.FileSet, err error) {
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

	for pPath := range packages {
		for j := range packages[pPath].Files {
			files = append(files, packages[pPath].Files[j])
		}
	}

	return path, files, fileSet, nil
}

func generate(ctx context.Context, loader loaderCallback, cfg Config) error {
	var (
		p  *loaderRecord
		ok bool
	)

	mut.Lock()
	if p, ok = records[cfg.Package]; !ok {
		p = &loaderRecord{
			Package:     nil,
			packageDefs: nil,
			files:       nil,
			targets:     []string{},
			path:        "",
			inited:      false,
			Mutex:       sync.Mutex{},
		}
		records[cfg.Package] = p
	}
	mut.Unlock()

	p.Lock()
	if !p.inited {
		var (
			err error
		)
		records[cfg.Package].path, records[cfg.Package].files, records[cfg.Package].fileSet, err = collectFiles(cfg.Package)
		if err != nil {
			return err
		}
		records[cfg.Package].Package, records[cfg.Package].packageDefs, err = initPackage(records[cfg.Package].path, records[cfg.Package].files, records[cfg.Package].fileSet)
		p.inited = true
	}
	for _, target := range records[cfg.Package].targets {
		if target == cfg.Structure {
			p.Unlock()
			return ProcessedError
		}
	}
	records[cfg.Package].targets = append(records[cfg.Package].targets, cfg.Structure)
	p.Unlock()

	for _, file := range records[cfg.Package].files {
		sf := newStructFinder(cfg.Structure, cfg.Package)
		ast.Inspect(file, sf.find)
		//todo check if struct is exist in package
		if sf.structure() != nil || len(sf.methods()) > 0 {
			gen := newGenerator(sf.imports(), cfg, loader, records[cfg.Package].path)
			if err := gen.Do(ctx, sf.structure(), sf.methods()); err != nil {
				log.Println(err)
				continue
			}
		}
	}

	return nil
}

func initPackage(path string, files []*ast.File, fs *token.FileSet) (*types.Package, packageDefs, error) {

	/**
	initializing package parsing with the go/type
	*/

	defs := make(map[*ast.Ident]types.Object)
	infos := &types.Info{
		Defs: defs,
	}

	config := types.Config{Importer: importer.Default(), FakeImportC: true}

	var err error
	pkg, err := config.Check(path, fs, files, infos)

	if err != nil {
		log.Println("Warning:", err)
		//return err
	}

	return pkg, defs, nil
}

func withLoader(ctx context.Context, cb loaderCallback) context.Context {
	return context.WithValue(ctx, "loader", cb)
}
