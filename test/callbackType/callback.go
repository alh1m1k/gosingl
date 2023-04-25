package callbackType

import (
	"fmt"
	"unsafe"
)

type typedCb func(pointer unsafe.Pointer, uintptr, uint16, call func(interface{})) (result map[string]any, err error)

type CallbackType struct {
	subfield1       int
	Subfield2       string
	privateFunction func() error
	PublicFunction  func(uintptr2 uintptr) error
	fmt.Stringer
}

func (receiver CallbackType) Empty() {}

func (receiver CallbackType) CallbackTyped(a, b, c typedCb, p **typedCb) (*typedCb, any, error) {
	return nil, nil, nil
}

func (receiver CallbackType) CallbackAnon(unsafe.Pointer, **typedCb, float32) {

}
