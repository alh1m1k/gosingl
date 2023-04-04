# Gosingle

gosingle generate module level singleton based on the chosen structure

## Installation

```sh
go get github.com/alh1m1k/gosingle
```

## Usage

With the following structure in the mystructure.go file :

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

just add the -w flag to write it to queryExecutor_singleton.go.

## Status

Package in alpha stage. Basic functionality, composition is not (yet) supported.
Implemented support for function fields. Zero test coverage.

## Dependencies

The two main dependencies are :

* [dave/jennifer](http://github.com/dave/jennifer): an awesome library for writing go code
* [mow.cli](http://github.com/jawher/mow.cli) : a cli command utility

# Based on
* [mrsinham/goreset](http://github.com/mrsinham/goreset): reset method generator for any structure


# TODO

- [ ] composition
- [ ] interface
- [ ] test coverage & testing