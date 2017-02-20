//go:generate go-bindata -o bin_data.go --pkg imager --nocompress --nometadata archive.png audio.png

package imager

// #cgo pkg-config: GraphicsMagick
// #cgo CFLAGS: -std=c11 -D_POSIX_C_SOURCE
// #include "init.h"
// #include "thumbnailer.h"
// #include <stdlib.h>
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	"meguca/config"
)

var (
	errTooWide             = errors.New("image too wide") // No such thing
	errTooTall             = errors.New("image too tall")
	errThumbnailingUnknown = errors.New("unknown thumbnailing error")
)

func init() {
	C.magickInit()
}

// UnloadGM safely unloads the GraphicsMagick runtime
func UnloadGM() {
	C.DestroyMagick()
}

// processImage generates a thumbnail from a source image buffer. If width and
// height are non-zero, buf is assumed to be a raw Y420 image.
// Returns the generated thumbnail's buffer, the source images and thumbnail's
// dimensions, if a the generated thumbnail is a PNG image, and error, if any.
func processImage(buf []byte, width, height uint) (
	[]byte, [4]uint16, bool, error,
) {
	src := C.struct_Buffer{
		data:   (*C.uint8_t)(C.CBytes(buf)),
		size:   C.size_t(len(buf)),
		width:  C.ulong(width),
		height: C.ulong(height),
	}
	defer C.free(unsafe.Pointer(src.data))

	var ex C.ExceptionInfo
	defer C.DestroyExceptionInfo(&ex)

	conf := config.Get()
	cOpts := C.struct_Options{
		JPEGCompression: C.uint8_t(conf.JPEGQuality),
		maxSrcWidth:     C.ulong(conf.MaxWidth),
		maxSrcHeight:    C.ulong(conf.MaxHeight),
	}

	var thumb C.struct_Thumbnail
	errCode := C.thumbnail(&src, &thumb, cOpts, &ex)
	defer func() {
		if thumb.img.data != nil {
			C.free(unsafe.Pointer(thumb.img.data))
		}
	}()
	var err error
	if ex.reason != nil {
		err = extractError(ex)
	} else {
		switch errCode {
		case 0:
		case 1:
			err = errThumbnailingUnknown
		case 2:
			err = errTooWide
		case 3:
			err = errTooTall
		}
	}
	if err != nil {
		return nil, [4]uint16{}, false, err
	}

	out := C.GoBytes(unsafe.Pointer(thumb.img.data), C.int(thumb.img.size))
	dims := [4]uint16{
		uint16(src.width),
		uint16(src.height),
		uint16(thumb.img.width),
		uint16(thumb.img.height),
	}
	return out, dims, bool(thumb.isPNG), nil
}

func extractError(ex C.ExceptionInfo) error {
	r := C.GoString(ex.reason)
	d := C.GoString(ex.description)
	return fmt.Errorf(`thumbnailer: %s: %s`, r, d)
}
