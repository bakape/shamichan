package imager

// #cgo pkg-config: GraphicsMagick
// #cgo CFLAGS: -std=c11
// #include "thumbnailer.h"
// #include <stdlib.h>
import "C"
import (
	"errors"
	"fmt"
	"os"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/imager/assets"
)

var (
	errTooWide = errors.New("image too wide") // No such thing
	errTooTall = errors.New("image too tall")
)

// InitImager applies the thumbnail quality configuration
func InitImager() error {
	C.InitializeMagick(C.CString(os.Args[0]))
	return assets.CreateDirs()
}

// processImage generates a thumbnail from a source image buffer. Returns the
// generated thumbnail's buffer, the source images and thumbnail's dimensions
// and error, if any.
func processImage(buf []byte) ([]byte, [4]uint16, error) {
	cBuf := C.CBytes(buf)
	defer C.free(cBuf)

	ex := &C.ExceptionInfo{}
	defer C.DestroyExceptionInfo(ex)

	conf := config.Get()
	var thumb C.struct_Thumbnail
	cOpts := C.struct_Options{
		JPEGCompression: C.uint8_t(conf.JPEGQuality),
		maxSrcWidth:     C.ulong(conf.MaxWidth),
		maxSrcHeight:    C.ulong(conf.MaxHeight),
	}
	errCode := C.thumbnail(cBuf, C.size_t(len(buf)), cOpts, &thumb, ex)
	defer func() {
		if thumb.buf != nil {
			C.free(thumb.buf)
		}
	}()
	var err error
	switch errCode {
	case 0:
	case 1:
		err = extractError(ex)
	case 2:
		err = errTooWide
	case 3:
		err = errTooTall
	}
	if err != nil {
		return nil, [4]uint16{}, err
	}

	out := C.GoBytes(thumb.buf, C.int(thumb.size))
	dims := [4]uint16{
		uint16(thumb.srcWidth),
		uint16(thumb.srcHeight),
		uint16(thumb.width),
		uint16(thumb.height),
	}
	return out, dims, nil
}

func extractError(ex *C.ExceptionInfo) error {
	r := C.GoString(ex.reason)
	d := C.GoString(ex.description)
	return fmt.Errorf(`thumbnailer: %s: %s`, r, d)
}
