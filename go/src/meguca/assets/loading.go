//go:generate go-bindata -o bin_data.go --pkg assets --nometadata -nocompress --prefix defaults defaults

package assets

// Stores board-specific loading images
var Loading = FileStore{
	m: make(map[string]File, 64),
	def: File{
		Data: MustAsset("loading.gif"),
		Mime: "image/gif",
	},
}
