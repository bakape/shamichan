/*
 Websocket server and client controler struct
*/

package server

import (
	"errors"
	"fmt"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/util"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

// uint8 identifiers for various message types
// 1 - 29 modify post model state
const (
	messageInvalid = iota
	messageInsertThread
	messageInsertPost
)

// >= 30 are miscelenious and do not write to post models
const (
	messageSynchronise = 30 + iota
	messageSwitchSync
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 5 * time.Second,
}

var textFrameReceived = websocket.CloseError{
	Code: websocket.CloseUnsupportedData,
	Text: "Only binary frames allowed",
}

func websocketHandler(res http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(res, req, nil)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(
			res,
			`Can only Upgrade to the Websocket protocol`,
			http.StatusBadRequest,
		)
		return
	} else if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	c := NewClient(conn)
	go c.receiverLoop()
	if err := c.listen(); err != nil {
		c.logError(err)
	}
}

// Client stores and manages a websocket-connected remote client and its
// interaction with the server and database
type Client struct {
	synced bool
	closed bool
	ident  auth.Ident
	sync.Mutex
	id       string
	conn     *websocket.Conn
	receiver chan []byte
	sender   chan []byte
	closer   chan websocket.CloseError
}

// NewClient creates a new websocket client
func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		id:       util.RandomID(32),
		ident:    auth.LookUpIdent(conn.RemoteAddr().String()),
		receiver: make(chan []byte),
		sender:   make(chan []byte),
		closer:   make(chan websocket.CloseError),
		conn:     conn,
	}
}

// listen listens for incoming messages on the Receiver, Sender and Closer
// channels and processes them sequentially
func (c *Client) listen() error {
	for c.isOpen() {
		select {
		case msg := <-c.closer:
			if !c.isOpen() { // Already closed. Just terminating this loop.
				return nil
			}
			return c.close(msg.Code, msg.Text)
		case msg := <-c.receiver:
			if err := c.receive(msg); err != nil {
				return err
			}
		case msg := <-c.sender:
			if err := c.send(msg); err != nil {
				return err
			}
		}
	}
	return nil
}

// Thread-safe way of checking, if the websocket connection is open
func (c *Client) isOpen() bool {
	c.Lock()
	defer c.Unlock()
	return !c.closed
}

// Set client to closed in a thread-safe way. Seperated for cleaner testing.
func (c *Client) setClosed() {
	c.Lock()
	c.closed = true
	c.Unlock()
}

// Convert the blocking websocket.Conn.ReadMessage() into a channel stream and
// handle errors
func (c *Client) receiverLoop() {
	for c.isOpen() {
		typ, message, err := c.conn.ReadMessage() // Blocking
		switch {
		case !c.isOpen(): // Closed, while waiting for message
			return
		case err != nil:
			return
		case typ != websocket.BinaryMessage:
			c.closer <- textFrameReceived
			return
		default:
			c.receiver <- message
		}
	}
}

// receive parses a message received from the client through websockets
func (c *Client) receive(msg []byte) error {
	if c.ident.Banned {
		return c.close(websocket.ClosePolicyViolation, "You are banned")
	}
	if len(msg) < 2 {
		return c.protocolError(msg)
	}
	typ := uint8(msg[0])
	if !c.synced && typ != messageSynchronise {
		return c.protocolError(msg)
	}

	data := msg[1:]
	var err error
	switch typ {
	case messageInsertThread:
		// TODO: Actual handlers
		fmt.Println(data)
	default:
		err = c.protocolError(msg)
	}
	return err
}

// protocolError handles malformed messages received from the client
func (c *Client) protocolError(msg []byte) error {
	errMsg := fmt.Sprintf("Invalid message: %s", msg)
	if err := c.close(websocket.CloseProtocolError, errMsg); err != nil {
		return util.WrapError{errMsg, err}
	}
	return errors.New(errMsg)
}

// logError writes the client's websocket error to the error log (or stdout)
func (c *Client) logError(err error) {
	log.Printf("Error by %s: %v\n", c.ident.IP, err)
}

// send sends a provided message as a websocket frame to the client
func (c *Client) send(msg []byte) error {
	return c.conn.WriteMessage(websocket.BinaryMessage, msg)
}

// Send thread-safely sends a message to the websocket client
func (c *Client) Send(msg []byte) {
	c.sender <- msg
}

// close closes a websocket connection with the provided status code and
// optional reason
func (c *Client) close(status int, reason string) error {
	err := c.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(status, reason),
		time.Now().Add(time.Second*5),
	)
	c.setClosed()
	close(c.closer) // Stop any looping select statements listening to this
	if err != nil {
		return err
	}
	return c.conn.Close()
}

// Close thread-safely closes the websocket connection with the supplied status
// code and optional reason string
func (c *Client) Close(status int, reason string) {
	c.closer <- websocket.CloseError{
		Code: status,
		Text: reason,
	}
}
