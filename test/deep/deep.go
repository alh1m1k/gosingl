package deep

import (
	"io"
)

type il1 interface {
	il2
	Ilvl1() error
}

type il2 interface {
	io.ByteScanner
	Ilvl2() error
}

type tl1 struct {
	tl2
}

type tl2 struct {
	io.SectionReader
}

type deep struct {
	il1
	tl1
	T0 func() error
}
