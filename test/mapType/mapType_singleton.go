// Code generated by <git repo>. DO NOT EDIT.
package mapType

var Instance *mapType

// <mapType> from github.com/alh1m1k/gosingl/test/mapType

func MapSimple(any2 any, bool2 bool, c int32, d map[string]string, g *string) (string, error) {
	return Instance.MapSimple(any2, bool2, c, d, g)
}

func MapType(any2 any, bool2 bool, c int32, d td, g *string) (string, error) {
	return Instance.MapType(any2, bool2, c, d, g)
}

func MapInterface(any2 any, bool2 bool, c int32, d map[string]interface {
	doStuff(a string) error
}, g *string) (string, error) {
	return Instance.MapInterface(any2, bool2, c, d, g)
}

func MapDeepWithTyped(any2 any, bool2 bool, c int32, z map[newFloat]map[any]callback, g *string) (string, error) {
	return Instance.MapDeepWithTyped(any2, bool2, c, z, g)
}

func MapWithSlice(z map[newFloat][]string) (string, error) {
	return Instance.MapWithSlice(z)
}

func MapWithArray(z map[complex128][6]string) (string, error) {
	return Instance.MapWithArray(z)
}

func MapWithChan(z map[string]chan bool) (string, error) {
	return Instance.MapWithChan(z)
}

func MapWithChanDirect1(z map[string]chan<- bool) (string, error) {
	return Instance.MapWithChanDirect1(z)
}

func MapWithChanDirect2(z map[string]<-chan int64) (string, error) {
	return Instance.MapWithChanDirect2(z)
}

func MapWithChanDirect3(z map[string]directChan) (string, error) {
	return Instance.MapWithChanDirect3(z)
}

func MapWithStruct(z map[string]struct {
	Field1, otherField int
	field2             callback
	Field3             any
}) (string, error) {
	return Instance.MapWithStruct(z)
}

func MapWithANY(z map[string]any) (string, error) {
	return Instance.MapWithANY(z)
}
