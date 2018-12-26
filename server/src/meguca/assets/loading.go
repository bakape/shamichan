//go:generate go-bindata -o bin_data.go --pkg assets --nometadata -nocompress --prefix files files

package assets

import (
	"meguca/util"
)

// Loading stores board-specific loading images
var Loading = FileStore{
	m: make(map[string]File, 64),
	def: File{
		Data: MustAsset("loading.gif"),
		Hash: util.HashBuffer(MustAsset("loading.gif")),
		Mime: "image/gif",
	},
}
