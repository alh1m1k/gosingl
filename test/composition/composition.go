package composition

import (
	"crypto"
	"fmt"
	"net/http"
	"os"
	"sync"
)

type composition struct {
	subfield1 int
	Subfield2 string
	fmt.State
	sync.Mutex `singl:"ignore"`
	os.Signal
	http.Client
	http.Server
	http.Response
	crypto.Decrypter
}
