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
	"errors"
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

// Various predefined thumbnailing errors
var (
	ErrTooWide = errors.New("thumbnailer: image too wide")
	ErrTooTall = errors.New("thumbnailer: image too tall")
)

// Options for thumbnailing a specific file
type Options struct {
	OutputType OutputType

	// Thumbnail dims
	Width, Height uint

	// Maximum allowed source image dimensions. Returns error, if exceeded.
	// Validation will not be conducted, if unset.
	MaxSrcWidth, MaxSrcHeight uint

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
		maxSrcWidth:     C.ulong(opts.MaxSrcWidth),
		maxSrcHeight:    C.ulong(opts.MaxSrcHeight),
	}
	err := C.thumbnail(cBuf, C.size_t(len(buf)), cOpts, &thumb, ex)
	defer func() {
		if thumb.buf != nil {
			C.free(thumb.buf)
		}
	}()
	switch err {
	case 0:
	case 1:
		return nil, 0, 0, extractError(ex)
	case 2:
		return nil, 0, 0, ErrTooWide
	case 3:
		return nil, 0, 0, ErrTooWide
	}

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
