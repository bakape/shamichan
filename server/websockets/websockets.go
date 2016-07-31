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
	return u.Host == config.AllowedOrigin
}

// Handler is an http.HandleFunc that responds to new websocket connection
// requests.
func Handler(res http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		ip := auth.GetIP(req)
		log.Printf("Error upgrading to websockets: %s: %s\n", ip, err)
		return
	}

	c := newClient(conn, req)
	if err := c.Listen(); err != nil {
		c.logError(err)
	}
}

// Client stores and manages a websocket-connected remote client and its
// interaction with the server and database
type Client struct {
	synced bool // to any change feed and the global Clients map
	auth.Ident
	conn         *websocket.Conn
	ID           string
	sessionToken string // Token of an authenticated user session, if any

	// Internal message receiver channel
	receive chan receivedMessage

	// Thread-safely sends a message to the websocket client
	write chan []byte

	// Close the client and free all used resources
	close chan struct{}

	// Close the update feed of a thread or board
	closeUpdateFeed chan struct{}
}

type receivedMessage struct {
	typ int
	msg []byte
	err error
}

// newClient creates a new websocket client
func newClient(conn *websocket.Conn, req *http.Request) *Client {
	return &Client{
		Ident:   auth.LookUpIdent(req),
		write:   make(chan []byte),
		close:   make(chan struct{}),
		receive: make(chan receivedMessage),
		conn:    conn,
	}
}

// Listen listens for incoming messages on the channels and processes them
func (c *Client) Listen() error {
	go c.receiverLoop()

	// Clean up, when loop exits
	err := c.listenerLoop()
	Clients.Remove(c.ID)
	return c.closeConnections(err)
}

// Separate function to ease error handling of the intenal client loop
func (c *Client) listenerLoop() error {
	ping := time.Tick(pingTimer)

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
		case msg := <-c.write:
			if err := c.send(msg); err != nil {
				return err
			}
		case <-ping:
			err := c.conn.WriteControl(
				websocket.PingMessage,
				pingMessage,
				time.Now().Add(writeTimeout),
			)
			if err != nil {
				return err
			}
		}
	}
}

// Close all conections an goroutines asociated with the Client
func (c *Client) closeConnections(err error) error {
	// Close client and update feed
	if c.closeUpdateFeed != nil {
		close(c.closeUpdateFeed)
	}
	if err != nil {
		close(c.close)
	}
	close(c.write)

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
func encodeMessage(typ messageType, msg interface{}) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	encoded := make([]byte, len(data)+2)
	typeString := strconv.FormatUint(uint64(typ), 10)

	// Ensure type string is always 2 chars long
	if len(typeString) == 1 {
		encoded[0] = '0'
		encoded[1] = typeString[0]
	} else {
		copy(encoded, typeString)
	}

	copy(encoded[2:], data)

	return encoded, nil
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

	for {
		msg := receivedMessage{}
		msg.typ, msg.msg, msg.err = c.conn.ReadMessage() // Blocking

		select {
		case <-c.close:
			return
		case c.receive <- msg:
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
			errMsg := append([]byte(err.Error()+": "), msg...)
			return errInvalidPayload(errMsg)
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
	log.Printf("Error by %s: %v\n", c.IP, err)
}

// Close closes a websocket connection with the provided status code and
// optional reason
func (c *Client) Close() {
	select {
	case <-c.close:
	default:
		close(c.close)
	}
}

// Small helper method for more DRY-ness. Not thread-safe.
func (c *Client) isLoggedIn() bool {
	return c.sessionToken != ""
}
