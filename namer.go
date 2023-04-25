package main

import (
	"fmt"
	"sync"
)

type namer struct {
	sync.Mutex
	m map[string]int64
	p []string
}

func (receiver *namer) New(typeOf string) string {
	receiver.Mutex.Lock()
	defer receiver.Mutex.Unlock()
	var v string
	if typeOf != "" {
		v = receiver.newNameOfType(typeOf)
	} else {
		v = receiver.newName()
	}
	receiver.p = append(receiver.p, v)
	return v
}

func (receiver *namer) Reset() {
	receiver.Mutex.Lock()
	defer receiver.Mutex.Unlock()
	receiver.m = make(map[string]int64)
	receiver.p = receiver.p[0:0]
}

func (receiver *namer) Len() int {
	return len(receiver.p)
}

func (receiver *namer) Values() []string {
	return receiver.p
}

func (receiver *namer) newNameOfType(typeOf string) string {
	var (
		indx int64
		ok   bool
	)
	if indx, ok = receiver.m[typeOf]; !ok {
		receiver.m[typeOf] = 0
	}
	receiver.m[typeOf]++
	return fmt.Sprintf("%s%d", typeOf, indx)
}

func (receiver *namer) newName() string {
	return receiver.newNameOfType("p")
}

func newNamer() *namer {
	n := &namer{}
	n.Reset()
	return n
}
