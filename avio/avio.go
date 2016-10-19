// Package avio contains helpers for creating libav contexts and passing
// io.ReadSeeker directly to ffmpeg.
package avio

// #cgo pkg-config: libavcodec libavutil libavformat
// #cgo CFLAGS: -std=c11
// #include "avio.h"
import "C"
import (
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

// MediaType cooresponds to the AVMediaType enum in ffmpeg. Exported separatly
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
	canRead = 1 << (iota + 1)
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
		m: make(map[uintptr]*Handlers),
	}

	// ErrFailedAVIOCtx indicates failure to create an AVIOContext
	ErrFailedAVIOCtx = errors.New("failed to initialize AVIOContext")

	// ErrFailedAVFCtx indicates faulure to create an AVFormatContext
	ErrFailedAVFCtx = errors.New("failed to initialize AVFormatContext")
)

// Handlers contains function prototypes for custom IO. Implement necessary
// prototypes and pass instance pointer to NewContext.
type Handlers struct {
	ReadPacket  func([]byte) (int, error)
	WritePacket func([]byte) (int, error)
	Seek        func(int64, int) (int64, error)
}

// Context is a wrapper for passing Go I/O interfaces to C
type Context struct {
	avFormatCtx *C.struct_AVFormatContext
	handlerKey  uintptr
}

type handlerMap struct {
	sync.RWMutex
	m map[uintptr]*Handlers
}

func (h *handlerMap) Set(k uintptr, handlers *Handlers) {
	h.Lock()
	h.m[k] = handlers
	h.Unlock()
}

func (h *handlerMap) Delete(k uintptr) {
	h.Lock()
	delete(h.m, k)
	h.Unlock()
}

func (h *handlerMap) Get(k unsafe.Pointer) *Handlers {
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
I/O psuedo-interface.

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
	this := &Context{}

	if handlers != nil {
		this.handlerKey = uintptr(unsafe.Pointer(ctx))
		handlersMap.Set(this.handlerKey, handlers)
	}

	var flags C.int
	if handlers.ReadPacket != nil {
		flags |= canRead
	}
	if handlers.WritePacket != nil {
		flags |= canWrite
	}
	if handlers.Seek != nil {
		flags |= canSeek
	}

	var err error
	this.avFormatCtx, err = C.format_context(ctx, C.int(IOBufferSize), flags)
	if err != nil {
		this.Close()
		return nil, err
	}
	if this.avFormatCtx == nil {
		this.Close()
		return nil, ErrFailedAVIOCtx
	}

	return this, nil
}

// Close closes and frees c. It should not be used after this point.
func (c *Context) Close() {
	if c.avFormatCtx != nil {
		C.destroy(c.avFormatCtx)
	}
	handlersMap.Delete(c.handlerKey)
}

// AVFormatContext returns the underlying AVFormatContext as an unsafe.Pointer
// for use in other packages.
func (c *Context) AVFormatContext() unsafe.Pointer {
	return unsafe.Pointer(c.avFormatCtx)
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
	codecCtx, err := C.codec_context(c.avFormatCtx, int32(typ))
	if err != nil {
		return nil, err
	}
	return unsafe.Pointer(codecCtx), nil
}

//export readCallBack
func readCallBack(opaque unsafe.Pointer, buf *C.uint8_t, bufSize C.int) C.int {
	handlers := handlersMap.Get(opaque)
	if handlers.ReadPacket == nil {
		panic("No reader handler initialized")
	}
	s := (*[1 << 30]byte)(unsafe.Pointer(buf))[:bufSize:bufSize]
	n, err := handlers.ReadPacket(s)
	if err != nil {
		return -1
	}
	return C.int(n)
}

//export writeCallBack
func writeCallBack(opaque unsafe.Pointer, buf *C.uint8_t, bufSize C.int) C.int {
	handlers := handlersMap.Get(opaque)
	if handlers.WritePacket == nil {
		panic("No writer handler initialized.")
	}

	n, err := handlers.WritePacket(C.GoBytes(unsafe.Pointer(buf), bufSize))
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
	handlers := handlersMap.Get(opaque)
	if handlers.Seek == nil {
		panic("No seek handler initialized.")
	}

	n, err := handlers.Seek(int64(offset), int(whence))
	if err != nil {
		return -1
	}
	return C.int64_t(n)
}
