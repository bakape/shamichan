// Package avio contains helpers for creating libav contexts and passing
// io.ReadSeeker directly to ffmpeg.
package avio

// #cgo pkg-config: libavcodec libavutil libavformat libswscale
// #cgo CFLAGS: -std=c11
// #include "avio.h"
import "C"
import (
	"errors"
	"fmt"
	"sync"
	"unsafe"
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
	avIOCtx     *C.struct_AVIOContext
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

	f, _ := os.Open(filepath.Join("testdata", "sample.mp4"))
	ctx, _ := NewContext(&Handlers{
		ReadPacket: f.Read,
		Seek:       f.Seek,
	})
	defer ctx.Free()

	avfc := ctx.AVFormatContext()
	... Do Something ...
*/
func NewContext(handlers *Handlers) (*Context, error) {
	ctx := C.avformat_alloc_context()
	this := &Context{}

	buffer := (*C.uchar)(C.av_malloc(C.size_t(IOBufferSize)))

	if buffer == nil {
		return nil, errors.New("unable to allocate buffer")
	}

	// we have to explicitly set it to nil, to force library using default
	// handlers
	var ptrRead, ptrWrite, ptrSeek *[0]byte = nil, nil, nil

	if handlers != nil {
		this.handlerKey = uintptr(unsafe.Pointer(ctx))
		handlersMap.Set(this.handlerKey, handlers)
	}

	if handlers.ReadPacket != nil {
		ptrRead = (*[0]byte)(C.readCallBack)
	}

	if handlers.WritePacket != nil {
		ptrWrite = (*[0]byte)(C.writeCallBack)
	}

	if handlers.Seek != nil {
		ptrSeek = (*[0]byte)(C.seekCallBack)
	}

	this.avIOCtx = C.avio_alloc_context(
		buffer,
		C.int(IOBufferSize),
		0,
		unsafe.Pointer(ctx),
		ptrRead,
		ptrWrite,
		ptrSeek,
	)
	if this.avIOCtx == nil {
		return nil, errors.New("unable to initialize avio context")
	}

	ctx.pb = this.avIOCtx
	if this.avFormatCtx = C.create_context(ctx); ctx == nil {
		this.Free()
		return nil, errors.New("failed to initialize AVFormatContext")
	}

	return this, nil
}

// Free frees up resources allocated for handling I/O in C
func (c *Context) Free() {
	handlersMap.Delete(c.handlerKey)
}

// AVIOContext returns the underlying AVIOContext as an unsafe.Pointer for use
// in other packages.
func (c *Context) AVIOContext() unsafe.Pointer {
	return unsafe.Pointer(c.avIOCtx)
}

// AVFormatContext returns the underlying AVFormatContext as an unsafe.Pointer
// for use in other packages.
func (c *Context) AVFormatContext() unsafe.Pointer {
	return unsafe.Pointer(c.avFormatCtx)
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
