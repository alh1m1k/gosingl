package split

func (receiver *split) InterfaceEmpty(any2 any, bool2 bool, c int32, d interface{}, g *string) (string, error) {
	return "", nil
}

func (receiver *split) InterfaceMethods(any2 any, bool2 bool, c int32, d interface {
	doStuff(a string) error
}, g *string) (string, error) {
	return "", nil
}

func (receiver *split) InterfaceMethodsDeep(any2 any, bool2 bool, c int32, d interface {
	IStuff(a string) error
	IStuff2(ifn interface {
		IStuffDeep(a string) error
	})
}, g *string) (string, error) {
	return "", nil
}
