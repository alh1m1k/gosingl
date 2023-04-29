// random comment
package callbackType

import (
	"context"
	"unsafe"
)

var cbInstance CallbackType

// <CallbackType> from github.com/alh1m1k/gosingl/test/callbackType

func Empty() {
	cbInstance.Empty()
}

func CallbackTyped(a, b, c typedCb, p **typedCb) (*typedCb, any, error) {
	return cbInstance.CallbackTyped(a, b, c, p)
}

func CallbackAnon(p0 unsafe.Pointer, p1 **typedCb, p2 float32, p3 interface {
	someStf(unsafe.Pointer, **typedCb, float32) func() error
}) {
	cbInstance.CallbackAnon(p0, p1, p2, p3)
}

func CallbackAnonNaming(p0 unsafe.Pointer, p1 **typedCb, p2 func(unsafe.Pointer, **typedCb, func(a, b, c int64) func(context.Context) error) error) {
	cbInstance.CallbackAnonNaming(p0, p1, p2)
}

func PublicFunction(uintptr2 uintptr) error {
	return cbInstance.PublicFunction(uintptr2)
}

// <fmt.Stringer> from fmt

func String() string {
	return cbInstance.Stringer.String()
}
