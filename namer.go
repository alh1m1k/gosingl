package main

import (
	"fmt"
	"sync"
)

type Namer interface {
	New(typeOf string) string
	Reset()
	Len() int
	Values() []string
	NewNamer() Namer
}

type ParameterNamer struct {
	sync.Mutex
	m map[string]int64
	p []string
}

func (receiver *ParameterNamer) New(typeOf string) string {
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

func (receiver *ParameterNamer) Reset() {
	receiver.Mutex.Lock()
	defer receiver.Mutex.Unlock()
	receiver.m = make(map[string]int64)
	receiver.p = receiver.p[0:0]
}

func (receiver *ParameterNamer) Len() int {
	return len(receiver.p)
}

func (receiver *ParameterNamer) Values() []string {
	return receiver.p
}

func (receiver *ParameterNamer) NewNamer() Namer {
	return newParameterNamer()
}

func (receiver *ParameterNamer) newNameOfType(typeOf string) string {
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

func (receiver *ParameterNamer) newName() string {
	return receiver.newNameOfType("p")
}

func newParameterNamer() *ParameterNamer {
	n := &ParameterNamer{}
	n.Reset()
	return n
}
