package assets

import (
	"io/ioutil"
	"path/filepath"

	"github.com/Chiiruno/meguca/static"
	"github.com/Chiiruno/meguca/util"
)

// Loading stores board-specific loading images
var Loading = FileStore{
	m: make(map[string]File, 64),
	def: File{
		Mime: "image/gif",
	},
}

func init() {
	err := func() (err error) {
		f, err := static.FS.Open(filepath.Join("/assets", "loading.gif"))
		if err != nil {
			return
		}
		defer f.Close()

		Loading.def.Data, err = ioutil.ReadAll(f)
		if err != nil {
			return
		}
		Loading.def.Hash = util.HashBuffer(Loading.def.Data)

		return
	}()
	if err != nil {
		panic(err)
	}
}
