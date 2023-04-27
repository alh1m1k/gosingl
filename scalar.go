package main

var scalar = [...]string{
	"int",
	"int8",
	"int16",
	"int32",
	"int64",
	"uint",
	"uint8",
	"uint16",
	"uint32",
	"uint64",
	"uintptr",
	"float32",
	"float64",
	"complex64",
	"complex128",
	"byte",
	"rune",
	"string",
	"boolean",
	"bool",
	"error",
	"chan",
	"any",
}

func ISScalarType(typeName string) bool {
	for _, str := range scalar {
		if str == typeName {
			return true
		}
	}
	return false
}
