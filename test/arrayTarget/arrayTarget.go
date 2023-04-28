package arrayTarget

import "unsafe"

type arrayTarget [2][3]rune

func (m arrayTarget) hided() {

}

func (m arrayTarget) Showed(pointer unsafe.Pointer, _ int, a, _, c bool) error {
	return nil
}
