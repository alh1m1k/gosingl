package main

import (
	"errors"
	"fmt"
	"go/ast"
	"sync"
)

var SignatureError = errors.New("wrong signature")
var AmbiguousError = errors.New("ambiguous")

type Checker interface {
	Valid(name string, in, out *ast.FieldList, isInterface bool, cfg Config) bool
	Reset()
	NewChecker() Checker
	Errors() []error
}

type uniqueCheckerRecord struct {
	name        string
	in, out     *ast.FieldList
	isInterface bool //interface not collide with interface or other rcv
}

type uniqueChecker struct {
	sync.Mutex
	index  []uniqueCheckerRecord
	errors map[error]int
}

func (u *uniqueChecker) Valid(name string, in, out *ast.FieldList, isInterface bool, cfg Config) bool {
	//todo opt lock
	for i := range u.index {
		if u.index[i].name == name {
			u.Lock()
			u.analyze(name, in, out, isInterface, &u.index[i])
			u.Unlock()
			return false
		}
	}
	u.Lock()
	defer u.Unlock()
	u.index = append(u.index, struct {
		name        string
		in, out     *ast.FieldList
		isInterface bool
	}{name: name, in: in, out: out, isInterface: isInterface})
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

func (u *uniqueChecker) analyze(name string, in, out *ast.FieldList, isInterface bool, record *uniqueCheckerRecord) {
	var (
		err error
	)
	if in.NumFields() == record.in.NumFields() || out.NumFields() == record.out.NumFields() {
		safe := isInterface || record.isInterface
		record.isInterface = record.isInterface && isInterface //reset if no
		if safe {
			return
		}
		err = fmt.Errorf("func %s was %w", name, AmbiguousError)

	} else {
		err = fmt.Errorf("func %s has entrys with %w", name, SignatureError)
	}
	if cnt, ok := u.errors[err]; !ok {
		u.errors[err] = 1
	} else {
		u.errors[err] = cnt + 1
	}
}

func newUniqueChecker() Checker {
	c := &uniqueChecker{}
	c.Reset()
	return c
}
