// Package websockets manages active websocket connections and messages received
// from and sent to them
package websockets

// #cgo CFLAGS: -std=c11
// #cgo LDFLAGS: -L${SRCDIR} -lwebsockets -ldl
// #include "bindings.h"
// #include <stdlib.h>
import "C"
import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/bakape/meguca/auth"
	"nhooyr.io/websocket"
)

// Client registry for binding with Rust. Needed, because Go pointers can
// not be stored in Rust.
var (
	clientsMu       sync.RWMutex
	clients         = make(map[uint64]client)
	clientIdCounter = uint64(0)

	errNonBinary = errors.New("non-binary message received")
)

// Client stores and manages a websocket-connected remote client and its
// interaction with the server and database
type client struct {
	// Remote IP of client
	ip net.IP

	// Used to receive from the client
	receive chan []byte

	// Used to send messages to the client
	send chan *C.WSMessage

	// Forcefully disconnect client with optional error.
	// This channel can receive a maximum of 2 messages during its lifetime,
	close chan error
}

// http.HandleFunc that responds to new websocket connection requests
func Handle(w http.ResponseWriter, r *http.Request) (err error) {
	// TODO: Pass IP to Rust
	ip, err := auth.GetIP(r)
	if err != nil {
		return
	}

	// TODO: Handle globally banned clients
	// // Prevents connection spam
	// err = db.IsBanned("all", ip)
	// if err != nil {
	// 	return
	// }

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	c := client{
		// Allows for ~60 seconds of messages, until the buffer overflows.
		// A larger gap is more acceptable to shitty connections and mobile
		// phones, especially while uploading.
		//
		// All calls to send must be non-blocking to reduce thread
		// contention.
		send: make(chan *C.WSMessage, (time.Second*60)/(time.Millisecond*100)),

		// This channel can receive a maximum of 2 messages during its lifetime,
		// so a buffer of 2 prevents any goroutine sending on this channel from
		// ever being blocked and leaking.
		close: make(chan error, 2),

		// Only ever called from one goroutine, so no bufferring needed
		receive: make(chan []byte),

		ip: ip,
	}

	id, err := register(c)
	if err != nil {
		return
	}
	defer unregister(id)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// TODO: defer unregister client

	go func() {
		var (
			w   bytes.Buffer
			typ websocket.MessageType
			r   io.Reader
			err error
		)
		for {
			typ, r, err = conn.Reader(ctx)
			if err != nil {
				goto fail
			}
			if typ != websocket.MessageBinary {
				err = errNonBinary
				goto fail
			}

			w.Reset()
			_, err = io.Copy(&w, r)
			if err != nil {
				goto fail
			}

			// TODO: Synchronously pass message to Rust
		}

	fail:
		select {
		case <-ctx.Done():
		case c.close <- err:
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case err = <-c.close:
			if err != nil {
				s := err.Error()
				if len(s) > 125 { // Max close message length
					s = s[:125]
				}
				// Ignore the close error. We don't care, if the client actually
				// receives the close message.
				conn.Close(websocket.StatusProtocolError, s)
			}
			return
		case msg := <-c.send:
			err = conn.Write(
				ctx,
				websocket.MessageBinary,
				*(*[]byte)(
					unsafe.Pointer(
						&reflect.SliceHeader{
							Data: uintptr(unsafe.Pointer(msg.data)),
							Len:  int(msg.size),
							Cap:  int(msg.size),
						},
					),
				),
			)
			C.ws_unref_message(msg)
			if err != nil {
				return
			}
		}
	}
}

// Register client and return its ID
func register(c client) (id uint64, err error) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	// Account for counter overflow
try:
	clientIdCounter++
	id = clientIdCounter
	_, ok := clients[id]
	if ok {
		goto try
	}

	clients[id] = c

	ip := C.CString(c.ip.String())
	defer C.free(unsafe.Pointer(ip))
	errC := C.ws_register_client(C.uint64_t(id), ip)
	defer C.free(unsafe.Pointer(errC))
	if errC != nil {
		err = errors.New(C.GoString(errC))
	}
	return
}

// Unregister client by ID
func unregister(id uint64) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	_, ok := clients[id]
	if ok {
		delete(clients, id)
		C.ws_unregister_client(C.uint64_t(id))
	}
}

//export ws_write_message
func ws_write_message(clientID C.uint64_t, msg *C.WSMessage) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	c, ok := clients[uint64(clientID)]
	if ok {
		c.send <- msg
	}
}

//export ws_close_client
func ws_close_client(clientID C.uint64_t, err *C.char) {
	if err != nil {
		defer C.free(unsafe.Pointer(err))
	}

	clientsMu.Lock()
	defer clientsMu.Unlock()

	c, ok := clients[uint64(clientID)]
	if ok {
		var e error
		if err != nil {
			e = errors.New(C.GoString(err))
		}
		c.close <- e

		// Make sure close is only ever written to once by the Rust bindings.
		// The client must be immediately unregistered after a close message has
		// been sent. The deferred unregister in the handler should NOP after
		// this.
		delete(clients, uint64(clientID))
	}
}
