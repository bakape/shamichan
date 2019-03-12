// Static file storage embedded into the binary

//go:generate statik --src ./src -f

package static

import (
	"net/http"

	_ "github.com/bakape/meguca/static/statik"
	"github.com/rakyll/statik/fs"
)

var (
	// Embedded in-binary filesystem. Contained files must not be modified.
	FS http.FileSystem
)

func init() {
	var err error
	FS, err = fs.New()
	if err != nil {
		panic(err)
	}
}

// Read file from embedded file system into buffer
func ReadFile(path string) (buf []byte, err error) {
	return fs.ReadFile(FS, path)
}
