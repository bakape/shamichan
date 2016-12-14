// Package thumbnailer provides a more efficient image thumbnailer than
// available with native Go image processing libraries through GraphicsMagic
// bindings.
package thumbnailer

// #cgo pkg-config: GraphicsMagick
// #cgo CFLAGS: -std=c11
// #include <stdlib.h>
// #include <magick/api.h>
// #include "thumbnailer.h"
import "C"
import (
	"fmt"
	"os"
)

func init() {
	C.InitializeMagick(C.CString(os.Args[0]))
}

// SetOpts sets global thumbnailer options
func SetOpts(maxX, maxY uint) {
	C.maxX = C.ulong(maxX)
	C.maxY = C.ulong(maxY)
}

// Thumbnail generates a thumbnail of the specified maximum dimensions from a
// source image buffer. jpeg specifies, ig the output thumbnail should be jpeg
// or PNG.
func Thumbnail(buf []byte, jpeg bool) ([]byte, uint, uint, error) {
	cBuf := C.CBytes(buf)
	defer C.free(cBuf)

	ex := &C.ExceptionInfo{}
	defer C.DestroyExceptionInfo(ex)

	var thumb C.struct_Thumbnail
	err := C.thumbnail(cBuf, C.size_t(len(buf)), &thumb, C.bool(jpeg), ex)
	if err != 0 {
		return nil, 0, 0, extractError(ex)
	}
	defer func() {
		if thumb.buf != nil {
			C.free(thumb.buf)
		}
	}()

	return C.GoBytes(thumb.buf, C.int(thumb.size)),
		uint(thumb.width),
		uint(thumb.height),
		nil
}

func extractError(ex *C.ExceptionInfo) error {
	r := C.GoString(ex.reason)
	d := C.GoString(ex.description)
	return fmt.Errorf(`thumbnailer: %s: %s`, r, d)
}
