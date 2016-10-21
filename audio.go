package goffmpeg

// #cgo pkg-config: libavcodec libavformat libavutil
// #cgo CFLAGS: -std=c11
// #include "audio.h"
import "C"
import "unsafe"

// HasImage return whether or not the file has album art in it
func (c *Context) HasImage() bool {
	return C.find_cover_art(c.avFormatCtx) != -1
}

// Picture extracts attached image. This function will only work if the decoder
// was given enough data.
func (c *Context) Picture() []byte {
	img := C.retrieve_cover_art(c.avFormatCtx)
	if img.size <= 0 || img.data == nil {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(img.data), img.size)
}
