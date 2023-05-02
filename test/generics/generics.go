package generics

import "os"

type ifloat interface {
	~float32 | ~float64
}

type iint interface {
	int | int64 | int32
}

type structure interface {
	*os.File | string
}

type innerGenerics[F iint, Z string | bool] struct {
}

type overlapGenerics[T comparable, G string | bool | float64] struct {
}

type longType[T1 comparable, T2 ifloat, T3 any, T4 structure] struct {
}

/*type wrongGenerics[T comparable, R bool | float32] struct {
}*/

type generics[T comparable, R string | bool | float64, Z structure] struct {
	innerGenerics[int, bool]
	overlapGenerics[float32, R]
	*longType[T, float32, Z, Z]
	//wrongGenerics[float32, R]
}

var gen generics[int, string, *os.File]

func (receiver generics[T, R, Z]) Typed(a T, b R) {

}

func (receiver innerGenerics[F, _]) InnerTyped(a F) {

}

func (receiver overlapGenerics[T, G]) OverlapTyped(a T) {

}

func (receiver *longType[T1, T2, T3, T4]) LongCall(a T1, b T2, c T3, d T4) {

}
