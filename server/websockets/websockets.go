// Package websockets manages active websocket connections and messages received
// from and sent to them.
package websockets

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	"github.com/gorilla/websocket"
)

// Overridable for faster testing
var (
	readTimeout  = time.Second * 30
	writeTimeout = time.Second * 30
)

// integer identifiers for various message types
// 1 - 29 modify post model state
const (
	messageInvalid = iota
	messageInsertThread
	messageInsertPost
)

// >= 30 are miscelenious and do not write to post models
const (
	messageSynchronise = 30 + iota
	messageResynchronise
	messageSwitchSync
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 5 * time.Second,
	CheckOrigin:      CheckOrigin,
}

// CheckOrigin asserts the client matches the origin specified by the server or
// has none.
func CheckOrigin(req *http.Request) bool {
	origin := req.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return u.Host == config.Get().HTTP.Origin
}

// Handler is an http.HandleFunc that responds to new websocket connection
// requests.
func Handler(res http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Printf(
			"Error upgrading to websockets: %s: %s\n",
			req.RemoteAddr,
			err,
		)
		return
	}

	c := newClient(conn)
	if err := c.Listen(); err != nil {
		c.logError(err)
	}
}

// Client stores and manages a websocket-connected remote client and its
// interaction with the server and database
type Client struct {
	synced bool
	ident  auth.Ident
	conn   *websocket.Conn
	ID     string

	// Internal message receiver channel
	receive chan receivedMessage

	// Send thread-safely sends a message to the websocket client
	Send chan []byte

	// Close the client and free all  used resources
	close chan error

	// Close the currently subscribed to update feed, if any
	closeFeed chan struct{}
}

type receivedMessage struct {
	typ int
	msg []byte
	err error
}

// newClient creates a new websocket client
func newClient(conn *websocket.Conn) *Client {
	return &Client{
		ident: auth.LookUpIdent(conn.RemoteAddr().String()),

		// Without buffering, a busy client would block the entire sender
		Send:    make(chan []byte, 1),
		close:   make(chan error, 1),
		receive: make(chan receivedMessage),
		conn:    conn,
	}
}

// Listen listens for incoming messages on the channels and processes them
func (c *Client) Listen() error {
	// Clean up, when loop exits
	defer Clients.Remove(c.ID)

	go c.receiverLoop()
	for {
		select {
		case <-c.close:
			return nil
		case msg := <-c.receive:
			if msg.err != nil {
				return msg.err
			}
			if err := c.handleMessage(msg.typ, msg.msg); err != nil {
				return err
			}
		case msg := <-c.Send:
			if err := c.send(msg); err != nil {
				return err
			}
		}
	}
}

// Sends a message to the client. Not safe for concurent use. Generally, you
// should be passing []byte to Client.Send instead.
func (c *Client) send(msg []byte) error {
	return c.conn.WriteMessage(websocket.TextMessage, msg)
}

// Format a mesage type as JSON and send it to the client. Not safe for
// concurent use.
func (c *Client) sendMessage(typ int, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	encoded := make([]byte, len(data)+2)
	typeString := strconv.Itoa(typ)

	// Ensure type string is always 2 chars long
	if len(typeString) == 1 {
		encoded[0] = '0'
		encoded[1] = typeString[0]
	} else {
		copy(encoded, typeString)
	}

	copy(encoded[2:], data)
	return c.send(encoded)
}

// receiverLoop proxies the blocking conn.ReadMessage() into the main client
// select loop.
func (c *Client) receiverLoop() {
	// Handle websocket timeout
	c.conn.SetReadDeadline(time.Now().Add(readTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})

	for {
		select {
		case <-c.close:
			return
		default:
			typ, msg, err := c.conn.ReadMessage() // Blocking
			if err != nil {
				select {
				case <-c.close:
				default:
					c.closeChannels()
				}
				return
			}
			c.receive <- receivedMessage{
				typ: typ,
				msg: msg,
				err: err,
			}
		}
	}
}

// Close channels to exit all running goroutines spawned for this client
func (c *Client) closeChannels() {
	close(c.close)
	if c.closeFeed != nil {
		close(c.closeFeed)
	}
}

// handleMessage parses a message received from the client through websockets
func (c *Client) handleMessage(msgType int, msg []byte) error {
	if msgType != websocket.TextMessage {
		return c.Close(
			websocket.CloseUnsupportedData,
			"Only text frames allowed",
		)
	}
	if c.ident.Banned {
		return c.Close(websocket.ClosePolicyViolation, "You are banned")
	}
	if len(msg) < 3 {
		return c.invalidPayload(msg)
	}

	// First two characters of a message define its type
	typ, err := strconv.Atoi(string(msg[:2]))
	if err != nil {
		return c.invalidPayload(msg)
	}
	if !c.synced && typ != messageSynchronise && typ != messageResynchronise {
		return c.invalidPayload(msg)
	}

	if err := c.runHandler(typ, msg); err != nil {
		switch err := err.(type) {
		case errInvalidMessage:
			return c.passError(msg, err)
		default:
			return err
		}
	}
	return nil
}

// Run the apropriate handler for the websocket message
func (c *Client) runHandler(typ int, msg []byte) error {
	data := msg[2:]
	switch typ {
	case messageSynchronise:
		return c.synchronise(data)
	case messageResynchronise:
		return c.resynchronise(data)
	default:
		return c.invalidPayload(msg)
	}
}

// invalidPayload handles malformed messages received from the client
func (c *Client) invalidPayload(msg []byte) error {
	return c.passError(msg, "Invalid message")
}

// Like invalidPayload, but allows passing a more detailed reason to the client.
func (c *Client) passError(msg []byte, reason interface{}) error {
	errMsg := fmt.Sprintf("%s: %s", reason, msg)
	err := c.Close(websocket.CloseInvalidFramePayloadData, errMsg)
	if err != nil {
		return util.WrapError(errMsg, err)
	}
	return errors.New(errMsg)
}

// logError writes the client's websocket error to the error log (or stdout)
// and closes the websocket connection, if not already closed.
func (c *Client) logError(err error) {
	log.Printf("Error by %s: %v\n", c.ident.IP, err)
	c.Close(websocket.CloseInternalServerErr, err.Error())
}

// Close closes a websocket connection with the provided status code and
// optional reason
func (c *Client) Close(status int, reason string) error {
	select {
	case <-c.close:
		return nil
	default:
		c.closeChannels()
		msg := websocket.FormatCloseMessage(status, reason)
		deadline := time.Now().Add(writeTimeout)
		return c.conn.WriteControl(websocket.CloseMessage, msg, deadline)
	}
}
