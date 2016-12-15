package imager

// #cgo pkg-config: libavcodec libavformat libavutil
// #cgo CFLAGS: -std=c11
// #include "audio.h"
import "C"
import (
	"io/ioutil"
	"path/filepath"
	"unsafe"
)

// Directory for static image asset storage. Overrideable for tests.
var assetRoot = "www"

// Fallback image for MP3 files with no cover
const fallbackCover = "audio-fallback.png"

// HasImage return whether or not the file has album art in it
func (c *ffContext) HasImage() bool {
	return C.find_cover_art(c.avFormatCtx) != -1
}

// Picture extracts attached image. This function will only work if the decoder
// was given enough data.
func (c *ffContext) Picture() []byte {
	img := C.retrieve_cover_art(c.avFormatCtx)
	if img.size <= 0 || img.data == nil {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(img.data), img.size)
}

// Test if file is an MP3
func detectMP3(buf []byte) (bool, error) {
	c, err := newFFContext(buf)
	if err != nil {
		// Invalid file that can't even have a context created
		if fferr, ok := err.(ffError); ok && fferr.Code() == -1 {
			return false, nil
		}
		return false, err
	}
	defer c.Close()
	codec, err := c.CodecName(ffAudio)
	if err != nil {
		return false, err
	}
	return codec == "mp3", nil
}

// Extract image and meta info from MP3 files and send them down the
// thumbnailing pipeline.
func processMP3(data []byte) (res thumbResponse) {
	c, err := newFFContext(data)
	if err != nil {
		res.err = err
		return
	}
	defer c.Close()
	res.length = uint32(c.Duration() / 1000000000)

	// No cover art in file. Assign fallback cover and return.
	if !c.HasImage() {
		return assignFallbackCover(res)
	}

	res.thumb, res.dims, res.err = processImage(c.Picture())
	return
}

// Assign fallback cover art to audio file without any
func assignFallbackCover(res thumbResponse) thumbResponse {
	path := filepath.Join(assetRoot, fallbackCover)
	res.thumb, res.err = ioutil.ReadFile(path)
	res.dims = [4]uint16{150, 150, 150, 150}
	return res
}
