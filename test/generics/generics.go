package generics

type innerGenerics[F int | float32, Z string | bool] struct {
}

type overlapGenerics[T comparable, G string | bool | float32] struct {
}

/*type wrongGenerics[T comparable, R bool | float32] struct {
}*/

type generics[T comparable, R string | bool | float64] struct {
	innerGenerics[int, bool]
	overlapGenerics[float32, R]
	//wrongGenerics[float32, R]
}

var gen generics[int, string]

func (receiver generics[T, R]) Typed(a T, b R) {

}

func (receiver innerGenerics[F, _]) InnerTyped(a F) {

}

func (receiver overlapGenerics[T, G]) OverlapTyped(a T) {

}
