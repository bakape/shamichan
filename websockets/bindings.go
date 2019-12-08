package websockets

// #include "bindings.h"
// #include <stdlib.h>
import "C"
import (
	"errors"
	"unsafe"

	"github.com/bakape/meguca/db"
)

//export ws_thread_exists
func ws_thread_exists(id C.uint64_t, err **C.char) (exists bool) {
	exists, _err := db.ThreadExists(uint64(id))
	if _err != nil {
		*err = C.CString(_err.Error())
	}
	return
}

//export ws_write_message
func ws_write_message(clientID C.uint64_t, msg *C.WSRcBuffer) {
	// Spawning separate goroutine to not block the pulsar thread pool
	go func() {
		// Not using deferred unlock to prevent possible deadlocks between the Go
		// and Rust client collection mutexes. These must be freed as soon as
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
	if err != nil {
		defer C.free(unsafe.Pointer(err))
	}

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
func ws_insert_thread() {

}
