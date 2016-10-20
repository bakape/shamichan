package goffmpeg

// #cgo pkg-config: libavcodec libavformat libavutil
// #cgo CFLAGS: -std=c11
// #include "audio.h"
import "C"
import "unsafe"

// AudioFormat returns the format string of the file's audio stream
func (c *Context) AudioFormat() string {
	ctx := C.get_codecContext(c.avFormatCtx)
	defer C.avcodec_close(ctx)
	if ctx == nil {
		return ""
	}
	return C.GoString(ctx.codec.name)
}

// Bitrate returns the bitrate in bits per second. For some files this will be
// absolute, for some an estimate.
func (c *Context) Bitrate() int64 {
	ctx := C.get_codecContext(c.avFormatCtx)
	defer C.avcodec_close(ctx)
	if ctx == nil || ctx.bit_rate == 0 {
		//This is an estimate, it might not be accurate!
		return int64(c.avFormatCtx.bit_rate)
	}
	return int64(ctx.bit_rate)
}

// HasImage return whether or not the file has album art in it
func (c *Context) HasImage() bool {
	return C.has_image(c.avFormatCtx) == 0
}

// Picture extracts attached image. This function will only work if the decoder
// was given enough data.
func (c *Context) Picture() []byte {
	img := C.retrieve_album_art(c.avFormatCtx)
	if img.size <= 0 || img.data == nil {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(img.data), img.size)
}
