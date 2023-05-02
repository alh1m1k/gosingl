package composition

import (
	"crypto"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

type innter struct {
	io.WriteCloser
}

type composition struct {
	subfield1 int
	Subfield2 string
	innter
	fmt.State
	sync.Mutex `singl:"ignore"`
	os.Signal
	http.Client
	http.Server
	http.Response
	crypto.Decrypter
	Skipped interface {
		Skipped() func()
	}
}
