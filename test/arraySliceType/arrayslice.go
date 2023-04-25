package arraySliceType

type _arraySlice_ struct {
	subfield1 int
	Subfield2 string
	Some      func(d interface {
		IStuff(a string) error
	})
}
