package websockets

// #cgo CFLAGS: -std=c11
// #cgo LDFLAGS: -ldl -lm
// #include "bindings.h"
// #include <stdlib.h>
import "C"
import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/bakape/meguca/auth"
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
	_exists, err := db.ThreadExists(uint64(id))
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
	tags []*C.char,
	tags_size C.size_t,
	auth_key *C.uint8_t,
	id *C.uint64_t,
) *C.char {
	_tags := make([]string, int(tags_size))
	for i := range _tags {
		_tags[i] = C.GoString(tags[i])
	}

	_id, err := db.InsertThread(
		C.GoString(subject),
		_tags,
		*(*auth.Token)(unsafe.Pointer(auth_key)),
	)
	if err != nil {
		return C.CString(err.Error())
	}
	*id = C.uint64_t(_id)
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

//export ws_get_feed_data
func ws_get_feed_data(id uint64) {
	go func() {
		// TODO: Read data from DB
		buf := []byte(fmt.Sprintf(`{"feed":%d}`, id))
		C.ws_receive_feed_data(C.uint64_t(id), toWSBuffer(buf), nil)
	}()
}

// Cast []bytes to WSBuffer without copy
func toWSBuffer(buf []byte) C.WSBuffer {
	h := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	return C.WSBuffer{
		(*C.uint8_t)(unsafe.Pointer(h.Data)),
		C.size_t(h.Len),
	}
}
