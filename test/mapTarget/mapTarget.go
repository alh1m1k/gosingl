package mapTarget

import "unsafe"

type mapTarget map[string]any

func (m mapTarget) hided() {

}

func (m mapTarget) Showed(pointer unsafe.Pointer, _ int, a, _, c bool) error {
	return nil
}
