// random comment
package callbackType

import "unsafe"

var cbInstance CallbackType

func Empty() {
	cbInstance.Empty()
}

func CallbackTyped(a, b, c typedCb, p **typedCb) (*typedCb, any, error) {
	return cbInstance.CallbackTyped(a, b, c, p)
}

func CallbackAnon(p0 unsafe.Pointer, p1 **typedCb, p2 float32) {
	cbInstance.CallbackAnon(p0, p1, p2)
}

func PublicFunction(uintptr2 uintptr) error {
	return cbInstance.PublicFunction(uintptr2)
}

// <&{fmt Stringer}> from /usr/local/go/src/fmt

func GoString() string {
	return cbInstance.GoString()
}
