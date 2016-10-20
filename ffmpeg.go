// Package goffmpeg provides an interface to pass I/O between Go and ffmpeg as
// well as methods for detecting codec format and generating thumbnail images
// from audio and video input.
package goffmpeg

// #cgo pkg-config: libavcodec libavutil libavformat
// #cgo CFLAGS: -std=c11
// #include "ffmpeg.h"
import "C"
import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"time"
	"unsafe"
)

// MediaType cooresponds to the AVMediaType enum in ffmpeg. Exported separately
// to bypass cross-package boundries.
type MediaType int

// Correspond to AVMediaType
const (
	Video MediaType = iota
	Audio
)

// Flags for enabling diffrent callbacks in AVIOContext
const (
	//export canRead
	canRead = 1 << iota
	//export canWrite
	canWrite
	//export canSeek
	canSeek
)

var (
	// IOBufferSize defines the size of the buffer used for allocating the C
	// response from ffmpeg. Not safe to be changed concurently.
	IOBufferSize = 4096

	// Global map of AVIOHandlers. One handlers struct per format context.
	// Using ctx pointer address as a key.
	handlersMap = handlerMap{
		m: make(map[uintptr]Handlers),
	}

	// ErrFailedAVIOCtx indicates failure to create an AVIOContext
	ErrFailedAVIOCtx = errors.New("failed to initialize AVIOContext")

	// ErrFailedAVFCtx indicates faulure to create an AVFormatContext
	ErrFailedAVFCtx = errors.New("failed to initialize AVFormatContext")
)

func init() {
	C.av_register_all()
	C.avcodec_register_all()
}

// Handlers contains function prototypes for custom IO. Implement necessary
// prototypes and pass instance pointer to NewContext.
type Handlers struct {
	Read  func([]byte) (int, error)
	Write func([]byte) (int, error)
	Seek  func(int64, int) (int64, error)
}

// Context is a wrapper for passing Go I/O interfaces to C
type Context struct {
	avFormatCtx *C.struct_AVFormatContext
	handlerKey  uintptr
}

// C can not retain any pointers to Go memory after the cgo call returns. We
// still need a way to bind AVFormatContext insatnces to Go I/O functions. To do
// that we convert the AVFormatContext pointer to a uintptr and use it as a key
// to look up the respective handlers on each call to one of the 3 I/O
// callbacks.
type handlerMap struct {
	sync.RWMutex
	m map[uintptr]Handlers
}

func (h *handlerMap) Set(k uintptr, handlers Handlers) {
	h.Lock()
	h.m[k] = handlers
	h.Unlock()
}

func (h *handlerMap) Delete(k uintptr) {
	h.Lock()
	delete(h.m, k)
	h.Unlock()
}

func (h *handlerMap) Get(k unsafe.Pointer) Handlers {
	h.RLock()
	handlers, ok := h.m[uintptr(k)]
	h.RUnlock()
	if !ok {
		panic(fmt.Sprintf(
			"No handlers instance found, according to pointer: %v",
			k,
		))
	}
	return handlers
}

/*
NewContext constructs a new AVIOContext and AVFormatContext to use on the passed
I/O psuedo-interface

Example:

	f, _ := os.Open("my file.mp4")
	ctx, _ := NewContext(&Handlers{
		ReadPacket: f.Read,
		Seek:       f.Seek,
	})
	defer ctx.Free()

	... Do Something ...
*/
func NewContext(handlers *Handlers) (*Context, error) {
	ctx := C.avformat_alloc_context()
	this := &Context{
		avFormatCtx: ctx,
	}

	if handlers != nil {
		this.handlerKey = uintptr(unsafe.Pointer(ctx))
		handlersMap.Set(this.handlerKey, *handlers)
	}

	var flags C.int
	if handlers.Read != nil {
		flags |= canRead
	}
	if handlers.Write != nil {
		flags |= canWrite
	}
	if handlers.Seek != nil {
		flags |= canSeek
	}

	err := C.create_context(&this.avFormatCtx, C.int(IOBufferSize), flags)
	if err < 0 {
		this.Close()
		return nil, FormatError(int(err))
	}
	if this.avFormatCtx == nil {
		this.Close()
		return nil, ErrFailedAVIOCtx
	}

	return this, nil
}

// NewContextReader reads the entirety of r and returns a Context to operate on
// the read buffer
func NewContextReader(r io.Reader) (*Context, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return NewContextReadSeeker(bytes.NewReader(b))
}

// NewContextReadSeeker is a helper for creating a Context from an io.ReadSeeker
func NewContextReadSeeker(r io.ReadSeeker) (*Context, error) {
	return NewContext(&Handlers{
		Read: r.Read,
		Seek: r.Seek,
	})
}

// Close closes and frees c. It should not be used after this point.
func (c *Context) Close() {
	if c.avFormatCtx != nil {
		C.destroy(c.avFormatCtx)
	}
	handlersMap.Delete(c.handlerKey)
}

// CodecName returns codec name of the best stream type in the input with
// desired verbosity of the codec name
func (c *Context) CodecName(typ MediaType, detailed bool) (
	string, error,
) {
	name, err := C.codec_name(c.avFormatCtx, int32(typ), C._Bool(detailed))
	if err != nil {
		return "", err
	}
	if name == nil {
		return "", nil
	}
	defer C.free(unsafe.Pointer(name))
	return C.GoString(name), nil
}

// CodecContext returns the AVCodecContext of the best stream of the passed
// MediaType. It is the responsibility of the caller to cast to his local
// C type. check the codec context for nil pointers and free memory, when done.
func (c *Context) CodecContext(typ MediaType) (unsafe.Pointer, error) {
	var codecCtx *C.struct_AVCodecContext
	err := C.codec_context(&codecCtx, c.avFormatCtx, int32(typ))
	if err < 0 {
		return nil, FormatError(int(err))
	}
	return unsafe.Pointer(codecCtx), nil
}

// FormatError converst an ffmpeg error message to a string
func FormatError(code int) error {
	str := C.format_error(C.int(code))
	defer C.free(unsafe.Pointer(str))
	return fmt.Errorf("ffmpeg: %s", C.GoString(str))
}

//export readCallBack
func readCallBack(opaque unsafe.Pointer, buf *C.uint8_t, bufSize C.int) C.int {
	s := (*[1 << 30]byte)(unsafe.Pointer(buf))[:bufSize:bufSize]
	n, err := handlersMap.Get(opaque).Read(s)
	if err != nil {
		return -1
	}
	return C.int(n)
}

//export writeCallBack
func writeCallBack(opaque unsafe.Pointer, buf *C.uint8_t, bufSize C.int) C.int {
	n, err := handlersMap.
		Get(opaque).
		Write(C.GoBytes(unsafe.Pointer(buf), bufSize))
	if err != nil {
		return -1
	}
	return C.int(n)
}

//export seekCallBack
func seekCallBack(
	opaque unsafe.Pointer,
	offset C.int64_t,
	whence C.int,
) C.int64_t {
	n, err := handlersMap.Get(opaque).Seek(int64(offset), int(whence))
	if err != nil {
		return -1
	}
	return C.int64_t(n)
}

// Length returns the length of the video
func (c *Context) Length() (time.Duration, error) {
	return time.Duration(c.avFormatCtx.duration * 1000), nil
}
