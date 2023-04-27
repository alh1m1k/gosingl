package main

import (
	"errors"
	"fmt"
	"go/ast"
	"sync"
)

var SignatureError = errors.New("wrong signature")

type Checker interface {
	Valid(name string, in, out *ast.FieldList, cfg Config) bool
	Reset()
	NewChecker() Checker
	Errors() []error
}

type uniqueCheckerRecord struct {
	name    string
	in, out *ast.FieldList
}

type uniqueChecker struct {
	sync.Mutex
	index  []uniqueCheckerRecord
	errors map[error]int
}

func (u *uniqueChecker) Valid(name string, in, out *ast.FieldList, cfg Config) bool {
	//todo opt lock
	for _, candidate := range u.index {
		if candidate.name == name {
			u.Lock()
			u.analyze(name, in, out, candidate)
			u.Unlock()
			return false
		}
	}
	u.Lock()
	defer u.Unlock()
	u.index = append(u.index, struct {
		name    string
		in, out *ast.FieldList
	}{name: name, in: in, out: out})
	return true
}

func (u *uniqueChecker) Reset() {
	u.Lock()
	defer u.Unlock()
	u.index = u.index[0:0]
	u.errors = map[error]int{}
}

func (u *uniqueChecker) Errors() []error {
	result := make([]error, 0, len(u.errors))
	for err, cnt := range u.errors {
		result = append(result, fmt.Errorf("%w %d times", err, cnt))
	}
	return result
}

func (u *uniqueChecker) NewChecker() Checker {
	return newUniqueChecker()
}

func (u *uniqueChecker) analyze(name string, in, out *ast.FieldList, record uniqueCheckerRecord) {
	if in.NumFields() != record.in.NumFields() || out.NumFields() != record.out.NumFields() {
		error := fmt.Errorf("func %s has entrys with %w", name, SignatureError)
		if cnt, ok := u.errors[error]; !ok {
			u.errors[error] = 1
		} else {
			u.errors[error] = cnt + 1
		}
	}
}

func newUniqueChecker() Checker {
	c := &uniqueChecker{}
	c.Reset()
	return c
}
