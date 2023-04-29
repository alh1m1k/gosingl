package main

import (
	"errors"
	"go/types"
	"sync"
)

var SignatureError = errors.New("wrong signature")
var AmbiguousError = errors.New("ambiguous")

type Checker interface {
	Check(target *wrappedFunctionDeclaration) bool
	Valid() []*wrappedFunctionDeclaration
	Invalid() []*wrappedFunctionDeclaration
	NewChecker([]*wrappedFunctionDeclaration) Checker
}

type uniqueChecker struct {
	sync.Mutex
	index, invalid, implemented []*wrappedFunctionDeclaration
	errors                      map[error]int
}

func (u *uniqueChecker) Check(target *wrappedFunctionDeclaration) bool {
	u.Lock()
	defer u.Unlock()
	for i := range u.index {
		if u.index[i] == target {
			return true
		}
	}
	return false
}

func (u *uniqueChecker) Valid() []*wrappedFunctionDeclaration {
	u.Lock()
	defer u.Unlock()
	return u.index
}

func (u *uniqueChecker) Invalid() []*wrappedFunctionDeclaration {
	u.Lock()
	defer u.Unlock()
	return u.invalid
}

func (u *uniqueChecker) NewChecker(total []*wrappedFunctionDeclaration) Checker {
	return newUniqueChecker2(total)
}

func (u *uniqueChecker) run() {
	u.Lock()
	defer u.Unlock()
	u.reset()
	output := make([]*wrappedFunctionDeclaration, 0)
start:
	for i := range u.index {
		for j := range u.index {
			if i == j || u.index[i] == nil || u.index[j] == nil {
				continue
			}
			if u.index[i].Name == u.index[j].Name {
				if u.analyze(u.index[i], u.index[j]) {
					if u.index[i].IsInterface {
						u.implemented = append(u.implemented, u.index[i])
						u.index[i] = nil //interface has been implemented
						continue start
					}
				} else {
					u.invalid = append(u.invalid, u.index[i])
					continue start
				}
			}
		}
		output = append(output, u.index[i])
	}
	u.index = output
	//log.Println("ending filter ", len(u.index), len(u.invalid))
}

func (u *uniqueChecker) reset() {
	u.errors = map[error]int{}
}

func (u *uniqueChecker) analyze(target, record *wrappedFunctionDeclaration) bool {

	if target.IsInterface || record.IsInterface {
		return types.Identical(target.Signature.Type(), record.Signature.Type())
	}

	//err = fmt.Errorf("func %s was %w", name, AmbiguousError)
	//err = fmt.Errorf("func %s has entrys with %w", name, SignatureError)

	/*	if cnt, ok := u.errors[err]; !ok {
			u.errors[err] = 1
		} else {
			u.errors[err] = cnt + 1
		}*/

	return false
}

func newUniqueChecker2(total []*wrappedFunctionDeclaration) Checker {
	c := &uniqueChecker{
		index: total,
	}
	c.run()
	return c
}
