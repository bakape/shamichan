package websockets

// #cgo CFLAGS: -std=c11
// #cgo LDFLAGS: -ldl -lm
// #include "bindings.h"
// #include <stdlib.h>
// #include <string.h>
import "C"
import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"unsafe"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/cache"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
	"github.com/go-playground/log"
)

func init() {
	util.Hook("config.updated", func() error {
		c := config.Get()
		C.ws_set_config(C.WSConfig{
			captcha: C.bool(c.Captcha),
		})
		return nil
	})
}

// Initialize module. Must be run after DB is online.
func Init() (err error) {
	buf, err := db.GetFeedData()
	if err != nil {
		return
	}
	C.ws_init(toWSBuffer(buf))
	return
}

// Construct byte slice from C pointer and size
func toSlice(ptr *C.uint8_t, size C.size_t) []byte {
	return *(*[]byte)(
		unsafe.Pointer(
			&reflect.SliceHeader{
				Data: uintptr(unsafe.Pointer(ptr)),
				Len:  int(size),
				Cap:  int(size),
			},
		),
	)
}

//export ws_thread_exists
func ws_thread_exists(id C.uint64_t, exists *bool) *C.char {
	_exists, err := db.ThreadExists(context.Background(), uint64(id))
	if err != nil {
		return C.CString(err.Error())
	}
	*exists = _exists
	return nil
}

//export ws_write_message
func ws_write_message(clientID C.uint64_t, msg C.WSRcBuffer) {
	// Spawning separate goroutine to not block the pulsar thread pool
	go func() {
		// Not using deferred unlock to prevent possible deadlocks between the
		// Go and Rust client collection mutexes. These must be freed as soon as
		// possible.
		clientsMu.RLock()
		c, ok := clients[uint64(clientID)]
		clientsMu.RUnlock()

		if ok {
			select {
			case c.send <- msg:
			case <-c.ctx.Done():
				// Client is dead - need to unreference in its stead
				C.ws_unref_message(msg.src)
			}
		} else {
			// No client, so unreference immediately
			C.ws_unref_message(msg.src)
		}
	}()
}

//export ws_close_client
func ws_close_client(clientID C.uint64_t, err *C.char) {
	// Not using deferred unlock to not block on channel send
	clientsMu.Lock()
	c, ok := clients[uint64(clientID)]
	clientsMu.Unlock()

	if ok {
		var e error
		if err != nil {
			e = errors.New(C.GoString(err))
		}
		select {
		case c.close <- e:
		case <-c.ctx.Done():
		}
	}
}

//export ws_insert_thread
func ws_insert_thread(
	subject *C.char,
	tags **C.char,
	tags_size C.size_t,
	auth_key *C.uint8_t,
	id *C.uint64_t,
) *C.char {
	_tags := make([]string, int(tags_size))
	size := unsafe.Sizeof(subject)
	for i := range _tags {
		_tags[i] = C.GoString(
			*(**C.char)(unsafe.Pointer(
				uintptr(unsafe.Pointer(tags)) + size*uintptr(i)),
			),
		)
	}

	_id, err := db.InsertThread(
		context.Background(),
		db.ThreadInsertParams{
			Subject: C.GoString(subject),
			Tags:    _tags,
			PostInsertParamsCommon: db.PostInsertParamsCommon{
				AuthKey: (*auth.AuthKey)(unsafe.Pointer(auth_key)),
			},
		},
	)
	if err != nil {
		return C.CString(err.Error())
	}
	*id = C.uint64_t(_id)

	cache.EvictThreadList()

	return nil
}

//export ws_validate_captcha
func ws_validate_captcha(
	id *C.uint8_t, // Always 64 bytes
	solution *C.uint8_t,
	size C.size_t,
) *C.char {
	// TODO: user-specific captchas after captchouli port to Postgres
	return nil
}

//export ws_log_error
func ws_log_error(err *C.char) {
	log.Errorf("websockets: %s", C.GoString(err))
}

// Cast []bytes to WSBuffer without copy
func toWSBuffer(buf []byte) C.WSBuffer {
	h := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	return C.WSBuffer{
		(*C.uint8_t)(unsafe.Pointer(h.Data)),
		C.size_t(h.Len),
	}
}

// Register image insertion into an open post.
//
// image: JSON-encoded inserted image data
func InsertImage(thread, post uint64, img common.Image) (err error) {
	buf, err := json.Marshal(img)
	if err != nil {
		return
	}
	return fromCError(C.ws_insert_image(
		C.uint64_t(thread),
		C.uint64_t(post),
		toWSBuffer(buf),
	))
}

// Cast any owned C error to Go and free it
func fromCError(errC *C.char) (err error) {
	if errC != nil {
		err = errors.New(C.GoString(errC))
	}
	C.free(unsafe.Pointer(errC))
	return
}
