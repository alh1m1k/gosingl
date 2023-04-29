package interfaceValidation

type i1 interface {
	Write(p []byte) (n int, err error)
}

type i2 interface {
	Write(p []byte, encoding uint8) (n int, err error)
}

type sub1 struct {
	i1
}

type interfaceValidationValid1 struct {
	sub1
	i1
	Write func(p []byte) (n int, err error)
}

type interfaceValidationInvalid1 struct {
	i1
	i2
}
