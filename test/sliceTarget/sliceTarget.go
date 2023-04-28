package sliceTarget

import "unsafe"

type sliceTarget []chan bool

func (m sliceTarget) hided() {

}

func (m sliceTarget) Showed(pointer unsafe.Pointer, _ func(_, _, _ float32), a, _, c bool) error {
	return nil
}
