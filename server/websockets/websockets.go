// Package websockets manages active websocket connections and messages received
// from and sent to them
package websockets

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/util"
	"github.com/gorilla/websocket"
)

const pingWriteTimeout = time.Second * 30

var (
	// Overrideable for faster tests
	pingTimer = time.Minute

	upgrader = websocket.Upgrader{
		HandshakeTimeout: 5 * time.Second,
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	}
)

// errInvalidPayload denotes a malformed messages received from the client
type errInvalidPayload []byte

func (e errInvalidPayload) Error() string {
	return fmt.Sprintf("invalid message: %s", string(e))
}

// errInvalidFrame denotes an invalid websocket frame in some other way than
// errInvalidMessage
type errInvalidFrame string

func (e errInvalidFrame) Error() string {
	return string(e)
}

// Client stores and manages a websocket-connected remote client and its
// interaction with the server and database
type Client struct {
	// Synchronized to any change feed and registered in the global Clients map.
	// Should only be mutated from Clients, which also contains weather this
	// Client is synced. The local property exists mainly to reduce lock
	// contention on Clients.
	synced bool

	// Client identity information
	auth.Ident

	// Post currently open by the client
	openPost openPost

	// Currently subscribed to update feed, if any
	feedID int64

	// Underlying websocket connection
	conn *websocket.Conn

	// Token of an authenticated user session, if any
	sessionToken string

	// Internal message receiver channel
	receive chan receivedMessage

	// Only used to pass messages from the Send method.
	sendExternal chan []byte

	// Close the client and free all used resources
	close chan error
}

type receivedMessage struct {
	typ int
	msg []byte
}

// Data of a post currently being written to by a Client
type openPost struct {
	hasImage bool
	bytes.Buffer
	bodyLength   int
	id, op, time int64
	board        string
}

// Handler is an http.HandleFunc that responds to new websocket connection
// requests.
func Handler(res http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		ip := auth.GetIP(req)
		log.Printf("websockets: %s: %s\n", ip, err)
		return
	}

	c := newClient(conn, req)
	if err := c.listen(); err != nil {
		c.logError(err)
	}
}

// newClient creates a new websocket client
func newClient(conn *websocket.Conn, req *http.Request) *Client {
	return &Client{
		Ident:   auth.LookUpIdent(req),
		close:   make(chan error, 2),
		receive: make(chan receivedMessage),
		// Allows for ~6 seconds of messages at 0.2 second intervals, until the
		// buffer overflows.
		sendExternal: make(chan []byte, 1<<5),
		conn:         conn,
	}
}

// Listen listens for incoming messages on the channels and processes them
func (c *Client) listen() error {
	go c.receiverLoop()

	// Clean up, when loop exits
	err := c.listenerLoop()
	Clients.remove(c)
	return c.closeConnections(err)
}

// Separate function to ease error handling of the internal client loop
func (c *Client) listenerLoop() error {
	// Periodically ping the client to ensure external proxies and CDNs do not
	// close the connection. Those have a tendency of sending 1001 to both ends
	// after rather short timeout, if no messages have been sent.
	ping := time.NewTicker(pingTimer)
	defer ping.Stop()

	for {
		select {
		case err := <-c.close:
			return err
		case msg := <-c.sendExternal:
			if err := c.send(msg); err != nil {
				return err
			}
		case <-ping.C:
			deadline := time.Now().Add(pingWriteTimeout)
			err := c.conn.WriteControl(websocket.PingMessage, nil, deadline)
			if err != nil {
				return err
			}
		case msg := <-c.receive:
			if err := c.handleMessage(msg.typ, msg.msg); err != nil {
				return err
			}
		}
	}
}

// Close all connections an goroutines associated with the Client
func (c *Client) closeConnections(err error) error {
	// Close update feed, if any
	if c.feedID != 0 {
		feeds.Remove <- subRequest{c.feedID, c}
		c.feedID = 0
	}

	// Close receiver loop
	c.Close(nil)

	// Send the client the reason for closing
	var closeType int
	switch err.(type) {
	case *websocket.CloseError:
		switch err.(*websocket.CloseError).Code {

		// Normal client-side websocket closure
		case websocket.CloseNormalClosure, websocket.CloseGoingAway:
			err = nil
			closeType = websocket.CloseNormalClosure

		// Ignore abnormal websocket closure as a network fault
		case websocket.CloseAbnormalClosure:
			err = nil
		}
	case nil:
		closeType = websocket.CloseNormalClosure
	default:
		c.sendMessage(MessageInvalid, err.Error())
		closeType = websocket.CloseInvalidFramePayloadData
	}

	// Try to send the client a close frame. This might fail, so ignore any
	// errors.
	if closeType != 0 {
		msg := websocket.FormatCloseMessage(closeType, "")
		deadline := time.Now().Add(time.Second)
		c.conn.WriteControl(websocket.CloseMessage, msg, deadline)
	}

	// Close socket
	closeError := c.conn.Close()
	if closeError != nil {
		err = util.WrapError(closeError.Error(), err)
	}

	return err
}

// Send a message to the client. Can be used concurrently.
func (c *Client) Send(msg []byte) {
	select {
	case c.sendExternal <- msg:
	default:
		c.Close(errors.New("send buffer overflow"))
	}
}

// Sends a message to the client. Not safe for concurrent use.
func (c *Client) send(msg []byte) error {
	return c.conn.WriteMessage(websocket.TextMessage, msg)
}

// Format a message type as JSON and send it to the client. Not safe for
// concurrent use.
func (c *Client) sendMessage(typ MessageType, msg interface{}) error {
	encoded, err := EncodeMessage(typ, msg)
	if err != nil {
		return err
	}
	return c.send(encoded)
}

// receiverLoop proxies the blocking conn.ReadMessage() into the main client
// select loop.
func (c *Client) receiverLoop() {
	for {
		var (
			err error
			msg receivedMessage
		)
		msg.typ, msg.msg, err = c.conn.ReadMessage() // Blocking
		if err != nil {
			c.Close(err)
			return
		}

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
		return errInvalidFrame("only text frames allowed")
	}
	if len(msg) < 2 {
		return errInvalidPayload(msg)
	}

	// First two characters of a message define its type
	uncast, err := strconv.ParseUint(string(msg[:2]), 10, 8)
	if err != nil {
		return errInvalidPayload(msg)
	}
	typ := MessageType(uncast)
	if !c.synced && typ != MessageSynchronise {
		return errInvalidPayload(msg)
	}

	return c.runHandler(typ, msg)
}

// Run the appropriate handler for the websocket message
func (c *Client) runHandler(typ MessageType, msg []byte) error {
	data := msg[2:]
	handler, ok := handlers[typ]
	if !ok {
		return errInvalidPayload(msg)
	}
	return handler(data, c)
}

// logError writes the client's websocket error to the error log (or stdout)
func (c *Client) logError(err error) {
	log.Printf("error by %s: %v\n", c.IP, err)
}

// Close closes a websocket connection with the provided status code and
// optional reason
func (c *Client) Close(err error) {
	select {
	case <-c.close:
	default:
		// Exit both for-select loops, if they have not exited yet
		for i := 0; i < 2; i++ {
			select {
			case c.close <- err:
			default:
			}
		}
	}
}

// Small helper method for more DRY-ness. Not thread-safe.
func (c *Client) isLoggedIn() bool {
	return c.sessionToken != ""
}

// Helper for determining, if the client currently has an open post not older
// than 29 minutes. If the post is older, it is closed automatically.
func (c *Client) hasPost() (bool, error) {
	switch {
	case c.openPost.id == 0:
		return false, errNoPostOpen
	case c.openPost.time < time.Now().Add(-time.Minute*29).Unix():
		return false, closePost(nil, c)
	default:
		return true, nil
	}
}
