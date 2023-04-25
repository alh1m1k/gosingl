package interfaceType

import (
	"fmt"
	"go/ast"
)

type td interface {
	typeStuff(a string) error
}
type callback func(td) (any, error)

type interfaceType struct {
	subfield1 int
	Subfield2 string
	Some      func(d interface {
		IStuff(a string) error
	})
}

func (receiver *interfaceType) InterfaceEmpty(any2 any, bool2 bool, c int32, d interface{}, g *string) (string, error) {
	return "", nil
}

func (receiver *interfaceType) InterfaceType(any2 any, bool2 bool, c int32, td, g *string) (string, error) {
	return "", nil
}

func (receiver *interfaceType) InterfaceMethods(any2, zomzom any, bool2 bool, c int32, d interface {
	doStuff(a string) error
	doStuffWithChan(a chan any) (<-chan int64, error)
	doStuffWithStruct(a chan any) (struct {
		ast.Field
		abc, def string
		callback
		_ map[string]any
		fmt.Stringer
		Include struct {
			diction string `gosing:"some"`
		}
	}, error)
}, g *string) (string, error) {
	return "", nil
}

func (receiver *interfaceType) InterfaceMethodsDeepAndCallback(any2 any, bool2 bool, c callback, d interface {
	IStuff(a string) error
	td
	IStuff2(ifn interface {
		IStuffDeep(a string) error
		td
	})
}, g *string) (string, error) {
	return "", nil
}
