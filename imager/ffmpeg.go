package imager

// #cgo pkg-config: libavcodec libavutil libavformat
// #cgo CFLAGS: -std=c11
// #include "ffmpeg.h"
import "C"
import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
	"unsafe"
)

// ffMediaType corresponds to the AVMediaType enum in ffmpeg
type ffMediaType int8

const (
	ffUnknown ffMediaType = iota - 1
	ffVideo
	ffAudio
)

var (
	// Global map of AVIOHandlers. One handlers struct per format context.
	// Using ctx pointer address as a key.
	handlersMap = handlerMap{
		m: make(map[uintptr]io.ReadSeeker),
	}

	errFailedAVFCtx = errors.New("failed to initialize AVFormatContext")
)

func init() {
	C.av_register_all()
	C.avcodec_register_all()
	C.av_log_set_level(16)
}

// ffContext is a wrapper for passing Go I/O interfaces to C
type ffContext struct {
	avFormatCtx *C.struct_AVFormatContext
	handlerKey  uintptr
	codecs      map[ffMediaType]codecInfo
}

// Container for allocated codecs, so we can reuse them
type codecInfo struct {
	stream C.int
	ctx    *C.AVCodecContext
}

// ffError converts an FFmpeg error code to a Go error with a human-readable
// error message
type ffError C.int

// C can not retain any pointers to Go memory after the cgo call returns. We
// still need a way to bind AVFormatContext instances to Go I/O functions. To do
// that we convert the AVFormatContext pointer to a uintptr and use it as a key
// to look up the respective handlers on each call.
type handlerMap struct {
	sync.RWMutex
	m map[uintptr]io.ReadSeeker
}

// Error formats the FFmpeg error in human-readable format
func (f ffError) Error() string {
	str := C.format_error(C.int(f))
	defer C.free(unsafe.Pointer(str))
	return fmt.Sprintf("ffmpeg: %s", C.GoString(str))
}

// Code returns the underlying FFmpeg error code
func (f ffError) Code() int {
	return int(f)
}

func (h *handlerMap) Set(k uintptr, rs io.ReadSeeker) {
	h.Lock()
	h.m[k] = rs
	h.Unlock()
}

func (h *handlerMap) Delete(k uintptr) {
	h.Lock()
	delete(h.m, k)
	h.Unlock()
}

func (h *handlerMap) Get(k unsafe.Pointer) io.ReadSeeker {
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

// newFFContext constructs a new AVIOContext and AVFormatContext
func newFFContext(buf []byte) (*ffContext, error) {
	ctx := C.avformat_alloc_context()
	this := &ffContext{
		avFormatCtx: ctx,
		codecs:      make(map[ffMediaType]codecInfo),
	}

	this.handlerKey = uintptr(unsafe.Pointer(ctx))
	handlersMap.Set(this.handlerKey, bytes.NewReader(buf))

	err := C.create_context(&this.avFormatCtx)
	if err < 0 {
		this.Close()
		return nil, ffError(err)
	}
	if this.avFormatCtx == nil {
		this.Close()
		return nil, errFailedAVFCtx
	}

	return this, nil
}

// Close closes and frees memory allocated for c. c should not be used after
// this point.
func (c *ffContext) Close() {
	for _, ci := range c.codecs {
		C.avcodec_free_context(&ci.ctx)
	}
	if c.avFormatCtx != nil {
		C.destroy(c.avFormatCtx)
	}
	handlersMap.Delete(c.handlerKey)
}

// Allocate a codec context for the best stream of the passed ffMediaType, if not
// allocated already
func (c *ffContext) codecContext(typ ffMediaType) (codecInfo, error) {
	if ci, ok := c.codecs[typ]; ok {
		return ci, nil
	}

	var (
		ctx    *C.struct_AVCodecContext
		stream C.int
	)
	err := C.codec_context(&ctx, &stream, c.avFormatCtx, int32(typ))
	if err < 0 {
		return codecInfo{}, ffError(err)
	}

	ci := codecInfo{
		stream: stream,
		ctx:    ctx,
	}
	c.codecs[typ] = ci
	return ci, nil
}

// CodecName returns the codec name of the best stream of type typ in the input
// or an empty string, if there is no stream of this type
func (c *ffContext) CodecName(typ ffMediaType) (string, error) {
	ci, err := c.codecContext(typ)
	if err == nil {
		return C.GoString(ci.ctx.codec.name), nil
	}
	fferr, ok := err.(ffError)
	if ok && C.int(fferr) == C.AVERROR_STREAM_NOT_FOUND {
		return "", nil
	}
	return "", err
}

// Duration returns the duration of the input
func (c *ffContext) Duration() time.Duration {
	return time.Duration(c.avFormatCtx.duration * 1000)
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
