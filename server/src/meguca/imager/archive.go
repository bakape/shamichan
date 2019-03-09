package imager

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/ulikunitz/xz"
)

const (
	mimeZip   = "application/zip"
	mime7Zip  = "application/x-7z-compressed"
	mimeTarGZ = "application/gzip"
	mimeTarXZ = "application/x-xz"
)

// Detect if file is a TAR archive compressed with GZIP
func detectTarGZ(buf []byte) (mime string, ext string) {
	if !bytes.HasPrefix(buf, []byte("\x1F\x8B\x08")) {
		return
	}

	r, err := gzip.NewReader(bytes.NewReader(buf))
	switch {
	case err != nil:
	case isTar(r):
		mime = mimeTarGZ
		ext = "tar.gz"
	}
	return
}

// Read the start of the file and determine, if it is a TAR archive
func isTar(r io.Reader) bool {
	head := make([]byte, 262)
	read, err := r.Read(head)
	if err != nil || read != 262 {
		return false
	}
	return bytes.HasPrefix(head[257:], []byte("ustar"))
}

// Detect if file is a TAR archive compressed with XZ
func detectTarXZ(buf []byte) (mime string, ext string) {
	if !bytes.HasPrefix(buf, []byte{0xFD, '7', 'z', 'X', 'Z', 0x00}) {
		return "", ""
	}

	r, err := xz.NewReader(bytes.NewReader(buf))
	switch {
	case err != nil:
	case isTar(r):
		mime = mimeTarXZ
		ext = "tar.xz"
	}
	return
}
