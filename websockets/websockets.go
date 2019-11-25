// Package websockets manages active websocket connections and messages received
// from and sent to them
package websockets

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/bakape/meguca/auth"
	"nhooyr.io/websocket"
)

// TODO: Client registry, which Rust uses for lookup

// Client stores and manages a websocket-connected remote client and its
// interaction with the server and database
type client struct {
	// Remote IP of client
	ip net.IP

	// Used to receive from the client
	receive chan []byte

	// Used to send messages to the client
	send chan []byte

	// Forcefully disconnect client with optional error.
	// This channel can receive a maximum of 2 messages during its lifetime,
	//
	// TODO: Make sure close is only ever written to once by the Rust binding.
	// The client must be immediately unregistered after a close message has
	// been sent. The deferred unregister in the handler should NOP after
	// this.
	close chan error
}

// Handler is an http.HandleFunc that responds to new websocket connection
// requests.
func Handler(w http.ResponseWriter, r *http.Request) (err error) {
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

	c := &client{
		// Allows for ~60 seconds of messages, until the buffer overflows.
		// A larger gap is more acceptable to shitty connections and mobile
		// phones, especially while uploading.
		//
		// All calls to send must be non-blocking to reduce thread
		// contention.
		send: make(chan []byte, (time.Second*60)/(time.Millisecond*100)),

		// This channel can receive a maximum of 2 messages during its lifetime,
		// so a buffer of 2 prevents any goroutine sending on this channel from
		// ever being blocked and leaking.
		close: make(chan error, 2),

		receive: make(chan []byte),
		ip:      ip,
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// TODO: Register client
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
				err = errors.New("non-binary message received")
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
			err = conn.Write(ctx, websocket.MessageBinary, msg)
			if err != nil {
				return
			}
		}
	}
}
