// Package websockets manages active websocket connections and messages received
// from and sent to them.
package websockets

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
	"github.com/gorilla/websocket"
)

// Overridable for faster testing
var (
	writeTimeout = time.Second * 30
	pingTimer    = time.Second * 30
	readTimeout  = time.Second * 40
	pingMessage  = []byte{1}
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 5 * time.Second,
	CheckOrigin:      CheckOrigin,
}

// errInvalidPayload denotes a malformed messages received from the client
type errInvalidPayload []byte

func (e errInvalidPayload) Error() string {
	return fmt.Sprintf("Invalid message: %s", string(e))
}

// errInvalidFrame denotes an invalid websocket frame in some other way than
// errInvalidMessage
type errInvalidFrame string

func (e errInvalidFrame) Error() string {
	return string(e)
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
	synced       bool // to any change feed and the global Clients map
	ident        auth.Ident
	conn         *websocket.Conn
	ID           string
	userID       string // ID an authenticated user, if currently logged in
	sessionToken string // Token of an authenticated user session, if any
	util.AtomicCloser
	updateFeedCloser *util.AtomicCloser

	// Internal message receiver channel
	receive chan receivedMessage

	// Send thread-safely sends a message to the websocket client
	Send chan []byte

	// Close the client and free all  used resources
	close chan error

	// AllocateImage receives Image structs from the thumbnailer for allocation
	// by the client
	AllocateImage chan types.Image
}

type receivedMessage struct {
	typ int
	msg []byte
}

// newClient creates a new websocket client
func newClient(conn *websocket.Conn) *Client {
	return &Client{
		ident:         auth.LookUpIdent(conn.RemoteAddr().String()),
		Send:          make(chan []byte),
		close:         make(chan error),
		receive:       make(chan receivedMessage),
		AllocateImage: make(chan types.Image),
		conn:          conn,
	}
}

// Listen listens for incoming messages on the channels and processes them
func (c *Client) Listen() (err error) {
	go c.receiverLoop()
	ping := time.Tick(pingTimer)

outer:
	for {
		select {
		case err = <-c.close:
			break outer
		case msg := <-c.receive:
			err = c.handleMessage(msg.typ, msg.msg)
			if err != nil {
				break outer
			}
		case msg := <-c.Send:
			err = c.send(msg)
			if err != nil {
				break outer
			}
		case <-ping:
			err = c.conn.WriteControl(
				websocket.PingMessage,
				pingMessage,
				time.Now().Add(writeTimeout),
			)
			if err != nil {
				break outer
			}
		case <-c.AllocateImage:

			// TODO: Image allocation

		}
	}

	// Clean up, when loop exits
	Clients.Remove(c.ID)
	return c.closeConnections(err)
}

// Close all conections an goroutines asociated with the Client
func (c *Client) closeConnections(err error) error {
	// Close client and update feed
	c.AtomicCloser.Close()
	if c.updateFeedCloser != nil {
		c.updateFeedCloser.Close()
	}
	close(c.Send)
	close(c.close)

	// Send the client the reason for closing
	var closeType int
	switch err.(type) {
	case errInvalidPayload, errInvalidFrame:
		c.sendMessage(messageInvalid, err.Error())
		closeType = websocket.CloseInvalidFramePayloadData
	case *websocket.CloseError:
		// Normal client-side websocket closure
		switch err.(*websocket.CloseError).Code {
		case websocket.CloseNormalClosure, websocket.CloseGoingAway:
			err = nil
			closeType = websocket.CloseNormalClosure
		}
	case nil:
		closeType = websocket.CloseNormalClosure
	default:
		closeType = websocket.CloseInternalServerErr
	}

	// Try to send the client a close frame. This might fail, so ignore any
	// errors.
	msg := websocket.FormatCloseMessage(closeType, "")
	deadline := time.Now().Add(time.Second)
	c.conn.WriteControl(websocket.CloseMessage, msg, deadline)

	// Close socket
	closeError := c.conn.Close()
	if closeError != nil {
		err = util.WrapError(closeError.Error(), err)
	}

	return err
}

// Sends a message to the client. Not safe for concurent use. Generally, you
// should be passing []byte to Client.Send instead.
func (c *Client) send(msg []byte) error {
	return c.conn.WriteMessage(websocket.TextMessage, msg)
}

// Format a mesage type as JSON and send it to the client. Not safe for
// concurent use.
func (c *Client) sendMessage(typ messageType, msg interface{}) error {
	encoded, err := encodeMessage(typ, msg)
	if err != nil {
		return err
	}
	return c.send(encoded)
}

// Encodes a message for sending through websockets. Separate function, so it
// can be used in tests.1
func encodeMessage(typ messageType, msg interface{}) (
	encoded []byte, err error,
) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	encoded = make([]byte, len(data)+2)
	typeString := strconv.FormatUint(uint64(typ), 10)

	// Ensure type string is always 2 chars long
	if len(typeString) == 1 {
		encoded[0] = '0'
		encoded[1] = typeString[0]
	} else {
		copy(encoded, typeString)
	}

	copy(encoded[2:], data)
	return
}

// receiverLoop proxies the blocking conn.ReadMessage() into the main client
// select loop.
func (c *Client) receiverLoop() {
	// Timeout connection, if no pongs received for 40 seconds
	c.conn.SetReadDeadline(time.Now().Add(readTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})

	for c.IsOpen() {
		typ, msg, err := c.conn.ReadMessage() // Blocking
		if err != nil {
			c.Close(err)
			break
		}
		c.receive <- receivedMessage{
			typ: typ,
			msg: msg,
		}
	}
}

// handleMessage parses a message received from the client through websockets
func (c *Client) handleMessage(msgType int, msg []byte) error {
	if msgType != websocket.TextMessage {
		return errInvalidFrame("Only text frames allowed")
	}
	if len(msg) < 3 {
		return errInvalidPayload(msg)
	}

	// First two characters of a message define its type
	uncast, err := strconv.ParseUint(string(msg[:2]), 10, 8)
	if err != nil {
		return errInvalidPayload(msg)
	}
	typ := messageType(uncast)
	if !c.synced && typ != messageSynchronise && typ != messageResynchronise {
		return errInvalidPayload(msg)
	}

	if err := c.runHandler(typ, msg); err != nil {
		switch err := err.(type) {
		case errInvalidMessage:
			return util.WrapError(err.Error(), errInvalidPayload(msg))
		default:
			return err
		}
	}
	return nil
}

// Run the apropriate handler for the websocket message
func (c *Client) runHandler(typ messageType, msg []byte) error {
	data := msg[2:]
	handler, ok := handlers[typ]
	if !ok {
		return errInvalidPayload(msg)
	}
	return handler(data, c)
}

// logError writes the client's websocket error to the error log (or stdout)
func (c *Client) logError(err error) {
	log.Printf("Error by %s: %v\n", c.ident.IP, err)
}

// Close closes a websocket connection with the provided status code and
// optional reason
func (c *Client) Close(err error) {
	if c.IsOpen() {
		c.close <- err
	}
}

// Small helper method for more DRY-ness. Not thread-safe.
func (c *Client) isLoggedIn() bool {
	return c.sessionToken != ""
}
