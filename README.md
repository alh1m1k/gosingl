# Gosingle

gosingle generate module level singleton based on the chosen structure or interface (map slice or array defined as type also supported).

## Note 
It's mostly machine translation. Edits are welcome.

## Installation

```sh
go get github.com/alh1m1k/gosingle
```

## Short
Generator utility to support specific package architecture

```go
package main

import "github.com/me/somepackage"

//default behavior
somepackage.Dosome1()
somepackage.Dosome2()

custom := somepackage.New(customCfg...)

//custom behavior
custom.Dosome1()
custom.Dosome2()
```

this architecture has the problem of having to duplicate the declaration of all public functions of the 
```somepackage.New()``` instance at package context. It is the problem of generating similar proxy 
functions that refer to ```somepackage.New()``` this generator is designed to solve.

## Usage

The generator does not generate any initialization code, only declaration.
Use module level init() or any other way to properly initialize singleton.

By default, generator recursively inspect target structure or interface as well as it composition members.
It seek for exported field function, exported methods and then wrap it with module level proxy function.

Global singleton variable may be hidden via lower case variable name. Variable type (ref or value) by default 
defined by target type (for interfaces, map slice and array) or rcv type of the first exported method and not 100% accurate.
Type may be set manualy fo example ```--variable *Instance``` Instance will be real or ```--variable &Instance```
Instance will be Ref. Also ```--variable Instance[int, float64]``` may be used to resolve generic type.

Composition member of target may be excluded from inspect via field tag ``` `singl:"ignore"'```

Filename for output file has template ```<package>/<target>_singleton.go``` and can be changed via ```--suffix``` or ```--filepath```.

File suffix ```_test.go``` and ```_singleton.go``` (value of ```--suffix``` flag)  are excluded from analyze.

Path to source files a resolved via ```build.Default.Import(pkg, ".", build.FindOnly)``` be careful with path and naming

All scalar type and all composite type are supported, as well as it's mixing and types decl are supported.

Generator will rename all ```_``` or anonymous function parameter to ```p0, p1 ...pn``` in order to generate valid function call 

With the following structure in the executor.go file :

```go
package queryExecutor

import (
	"context"
	"database/sql"
	"errors"
)

type QueryExecutor struct {
    One, Two string
}

// todo add opt contenxt param
func (receiver *QueryExecutor) Prepare(sqlStatement string) (context.Context, error) {
    return nil, nil
}

func (receiver *QueryExecutor) Query(sqlStatement any, params ...any) (*sql.Rows, error) {
    return nil, nil
}

func (receiver *QueryExecutor) Execute(sqlStatement any, params ...any) (sql.Result, error) {
    return nil, nil
}

func (receiver *QueryExecutor) WithSql(ctx context.Context, sqlStatement string, params ...any) context.Context {
    return nil
}

func (receiver *QueryExecutor) WithPrepare(ctx context.Context, sql *sql.Stmt) context.Context {
	return ctx
}

func (receiver *QueryExecutor) prepareContext(ctx context.Context) (*sql.Stmt, []any, error) {
    return nil, nil, nil
}

func (receiver *QueryExecutor) prepare(context context.Context, sqlStatement string) (*sql.Stmt, error) {
    return nil, nil
}

func (receiver *QueryExecutor) run(sqlStatement any, done finalizer, params ...any) (any, error) {
    return nil, nil
}
```

if you run :

```sh
$ gosingle github.com/me/queryExecutor QueryExecutor
```

you will have this output :

```go
// Code generated by <git repo>. DO NOT EDIT.
package queryExecutor

import (
	"context"
	"database/sql"
	"errors"
)

var Instance *QueryExecutor

// <QueryExecutor> from github.com/me/queryExecutor

func Prepare(sqlStatement string) (context.Context, error) {
	return Instance.Prepare(sqlStatement)
}

func Query(sqlStatement any, params ...any) (*sql.Rows, error) {
	return Instance.Query(sqlStatement, params...)
}

func Execute(sqlStatement any, params ...any) (sql.Result, error) {
	return Instance.Execute(sqlStatement, params...)
}

func WithSql(ctx context.Context, sqlStatement string, params ...any) context.Context {
	return Instance.WithSql(ctx, sqlStatement, params...)
}

func WithPrepare(ctx context.Context, sql *sql.Stmt) context.Context {
	return Instance.WithPrepare(ctx, sql)
}
```

just add the -w flag (or use other file related flag such as --suffix, --filepath) to write it to queryExecutor_singleton.go.

## Flags

	--PKG        package to walk to
	--TARGET     structure or interface that will be use as module singleton
	--variable   singleton instance (module variable) "Instance" by default
                 --variable lowercased "lowercased" will be unexported 
                 --variable &lowercased "lowercased" will be unexported and ref type
                 --variable *Uppercased "Uppercased" will be exported and real type
                 --variable &Uppercased[int, float64] "Uppercased" is exported ref variable of generic type
	--comment    code generated by <git repo>. DO NOT EDIT.
    --suffix     suffix of generated file "_singleton.go"
    --filepath   path for generated file --suffix will be ignored
	--deep       recursive deep

# Generics
Generic a not real case for this generator, I guess, it added mostly for reasons of completeness.

With the following structure in the generics.go file :

```go
package generics

import "os"

type structure interface {
	*os.File | string
}

type innerGenerics[F int | int64 | int32, Z string | bool] struct {
}

type overlapGenerics[T comparable, G string | bool | float64] struct {
}

type generics[T comparable, R string | bool | float64, Z structure] struct {
	innerGenerics[int, bool]
	overlapGenerics[float32, R]
}

var gen generics[int, string, *os.File]

func (receiver generics[T, R, Z]) Typed(a T, b R) {

}

func (receiver innerGenerics[F, _]) InnerTyped(a F) {

}

func (receiver overlapGenerics[T, G]) OverlapTyped(a T) {

}
```

if you run :

```sh
$ gosingl -w --variable "g[int, bool, *os.File]" github.com/me/generics generics
```
or
```sh
$ gosingl -w --variable "g[int, bool, &os.File]" github.com/me/generics generics
```

you will have this output :

```go
// Code generated by <git repo>. DO NOT EDIT.
package generics

import "os"

var g generics[int, bool, *os.File]

// <generics> from github.com/me/generics

func Typed(a int, b bool) {
	g.Typed(a, b)
}

func InnerTyped(a int) {
	g.innerGenerics.InnerTyped(a)
}

func OverlapTyped(a float32) {
	g.overlapGenerics.OverlapTyped(a)
}

func LongCall(a int, b float32, c *os.File, d *os.File) {
	g.longType.LongCall(a, b, c, d)
}
```

in context of resolving generic *os.File and &os.File are valid and produce *os.File as output.


## Status

Package in ready state. All basic functionality are implemented.
It has test coverage, but has lack of usage in real cases. So it can be broken, but not entirely :)
Generic has very basic support and not realy good checking, so generated code my have no sense.

## Dependencies

The two main dependencies are :

* [dave/jennifer](http://github.com/dave/jennifer): an awesome library for writing go code
* [mow.cli](http://github.com/jawher/mow.cli) : a cli command utility (probably will be dropped)

# Inspired by
* [mrsinham/goreset](http://github.com/mrsinham/goreset): reset method generator for any structure


# TODO

- [x] composition
- [x] interface
- [x] generics (basic)
- [x] tags (ignore)
- [x] test coverage & testing
- [x] ambiguous function detector
- [x] later variable decl
- [ ] ~~drop custom cli~~