package mapType

type td map[string]int64
type callback func(mapType) (td, newFloat, complex128, error)
type newFloat float32
type directChan chan<- string

type mapType struct {
	subfield1 int
	Subfield2 string
}

func (receiver *mapType) MapSimple(any2 any, bool2 bool, c int32, d map[string]string, g *string) (string, error) {
	return "", nil
}

func (receiver *mapType) MapType(any2 any, bool2 bool, c int32, d td, g *string) (string, error) {
	return "", nil
}

func (receiver *mapType) MapInterface(any2 any, bool2 bool, c int32, d map[string]interface {
	doStuff(a string) error
}, g *string) (string, error) {
	return "", nil
}

func (receiver *mapType) MapDeepWithTyped(any2 any, bool2 bool, c int32, z map[newFloat]map[any]callback, g *string) (string, error) {
	return "", nil
}

func (receiver *mapType) MapWithSlice(z map[newFloat][]string) (string, error) {
	return "", nil
}

func (receiver *mapType) MapWithArray(z map[complex128][6]string) (string, error) {
	return "", nil
}

func (receiver *mapType) MapWithChan(z map[string]chan bool) (string, error) {
	return "", nil
}

func (receiver *mapType) MapWithChanDirect1(z map[string]chan<- bool) (string, error) {
	return "", nil
}

func (receiver *mapType) MapWithChanDirect2(z map[string]<-chan int64) (string, error) {
	return "", nil
}

func (receiver *mapType) MapWithChanDirect3(z map[string]directChan) (string, error) {
	return "", nil
}

func (receiver *mapType) MapWithStruct(z map[string]struct {
	Field1, otherField int
	field2             callback
	Field3             any
}) (string, error) {
	return "", nil
}

func (receiver *mapType) MapWithANY(z map[string]any) (string, error) {
	return "", nil
}
