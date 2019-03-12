package imager

import (
	"unicode/utf8"
)

const mimeText = "text/plain"

// Detect any arbitrary text-like file
func detectText(buf []byte) (mime, ext string) {
	if utf8.Valid(buf) {
		return mimeText, ".txt"
	}
	return
}
