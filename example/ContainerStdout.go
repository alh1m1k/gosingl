// gosingl docker/compose/pkg/compose ContainerStdout
package compose

var Instance ContainerStdout

func Read(p []byte) (n int, err error) {
	return Instance.Read(p)
}

func Close() error {
	return Instance.Close()
}
