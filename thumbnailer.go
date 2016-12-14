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

// OutputType specifies output image formats
type OutputType int

// Available output formats
const (
	PNG OutputType = iota
	JPEG
)

// Options for thumbnailing a specific file
type Options struct {
	OutputType    OutputType
	Width, Height uint
	// Must be from interval [1;100]
	JPEGCompression uint
}

func init() {
	C.InitializeMagick(C.CString(os.Args[0]))
}

// Thumbnail generates a thumbnail of the specified maximum dimensions from a
// source image buffer. Returns the generated thumbnail buffer, the thumbnail's
// width and height and error, if any.
func Thumbnail(buf []byte, opts Options) ([]byte, uint, uint, error) {
	cBuf := C.CBytes(buf)
	defer C.free(cBuf)

	ex := &C.ExceptionInfo{}
	defer C.DestroyExceptionInfo(ex)

	var thumb C.struct_Thumbnail
	cOpts := C.struct_Options{
		outputType:      C.int(opts.OutputType),
		width:           C.ulong(opts.Width),
		height:          C.ulong(opts.Height),
		JPEGCompression: C.ulong(opts.JPEGCompression),
	}
	err := C.thumbnail(cBuf, C.size_t(len(buf)), cOpts, &thumb, ex)
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
