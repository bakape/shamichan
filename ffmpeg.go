// Package goffmpeg provides an interface to pass I/O between Go and ffmpeg as
// well as methods for detecting codec format and generating thumbnail images
// from audio and video input. Also provides an MKV, WEBM and MP4 driver for
// the "image" package.
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
	"unsafe"
)

// MediaType coresponds to the AVMediaType enum in ffmpeg. Exported separately
// to bypass cross-package boundries.
type MediaType int

// Correspond to AVMediaType
const (
	Unknown MediaType = iota - 1
	Video
	Audio
	Data
	Subtitle
	Attachment
	NB
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

	// ErrFailedAVFCtx indicates faulure to create an AVFormatContext
	ErrFailedAVFCtx = errors.New("failed to initialize AVFormatContext")
)

func init() {
	C.av_register_all()
	C.avcodec_register_all()
	C.av_log_set_level(16)
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
	codecs      map[MediaType]codecInfo
}

// Contaoiner for allocated coddecs, so we can reuse them
type codecInfo struct {
	stream C.int
	ctx    *C.AVCodecContext
}

// FFmpegError converst an FFmpeg error code to a Go error with a
// human-readable error message
type FFmpegError C.int

// C can not retain any pointers to Go memory after the cgo call returns. We
// still need a way to bind AVFormatContext insatnces to Go I/O functions. To do
// that we convert the AVFormatContext pointer to a uintptr and use it as a key
// to look up the respective handlers on each call to one of the 3 I/O
// callbacks.
type handlerMap struct {
	sync.RWMutex
	m map[uintptr]Handlers
}

// Error formats the FFmpeg error in human-readable format
func (f FFmpegError) Error() string {
	str := C.format_error(C.int(f))
	defer C.free(unsafe.Pointer(str))
	return fmt.Sprintf("ffmpeg: %s", C.GoString(str))
}

// Code returns the underlying FFmpeg error code
func (f FFmpegError) Code() int {
	return int(f)
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
		codecs:      make(map[MediaType]codecInfo),
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
		return nil, FFmpegError(err)
	}
	if this.avFormatCtx == nil {
		this.Close()
		return nil, ErrFailedAVFCtx
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

// Close closes and frees memory allocated for c. c should not be used after
// this point.
func (c *Context) Close() {
	for _, ci := range c.codecs {
		C.avcodec_free_context(&ci.ctx)
	}
	if c.avFormatCtx != nil {
		C.destroy(c.avFormatCtx)
	}
	handlersMap.Delete(c.handlerKey)
}

// Allocate a codec context for the best stream of the passed MediaType, if not
// allocated allready
func (c *Context) codecContext(typ MediaType) (codecInfo, error) {
	if ci, ok := c.codecs[typ]; ok {
		return ci, nil
	}

	var (
		ctx    *C.struct_AVCodecContext
		stream C.int
	)
	err := C.codec_context(&ctx, &stream, c.avFormatCtx, int32(typ))
	if err < 0 {
		return codecInfo{}, FFmpegError(err)
	}

	ci := codecInfo{
		stream: stream,
		ctx:    ctx,
	}
	c.codecs[typ] = ci
	return ci, nil
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
