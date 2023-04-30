package generics

type generics[T comparable, R any] struct {
}

var gen generics[int, bool]

func (receiver generics[T, R]) Typed(a T, b R) {

}
