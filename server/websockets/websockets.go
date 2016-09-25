// Package websockets manages active websocket connections and messages received
// from and sent to them.
package websockets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 5 * time.Second,
	CheckOrigin:      CheckOrigin,
}

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
	// Synchronised to any change feed and regstered in the global Clients map.
	// Should only be mutated from Clients, which also contains weather this
	// Client is synced. The local property exists mainly to reduce lock
	// contention on Clients.
	synced bool

	// Client identity information
	auth.Ident

	// Post currently open by the client
	openPost openPost

	// Protects c.send
	sendMu sync.Mutex

	// Currently subscribed to update feed, if any
	feed *updateFeed

	// Underlyting websocket connection
	conn *websocket.Conn

	// Token of an authenticated user session, if any
	sessionToken string

	// Internal message receiver channel
	receive chan receivedMessage

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
		log.Printf("websockets: %s: %s\n", ip, err)
		return
	}

	c := newClient(conn, req)
	if err := c.Listen(); err != nil {
		c.logError(err)
	}
}

// newClient creates a new websocket client
func newClient(conn *websocket.Conn, req *http.Request) *Client {
	return &Client{
		Ident:   auth.LookUpIdent(req),
		close:   make(chan error, 2),
		receive: make(chan receivedMessage),
		conn:    conn,
	}
}

// Listen listens for incoming messages on the channels and processes them
func (c *Client) Listen() error {
	go c.receiverLoop()

	// Clean up, when loop exits
	err := c.listenerLoop()
	Clients.Remove(c)
	return c.closeConnections(err)
}

// Separate function to ease error handling of the intenal client loop
func (c *Client) listenerLoop() error {
	for {
		select {
		case err := <-c.close:
			return err
		case msg := <-c.receive:
			if err := c.handleMessage(msg.typ, msg.msg); err != nil {
				return err
			}
		}
	}
}

// Request and receive any messages the client is behind on
func (c *Client) fetchBacklog(ctr int) {
	if c.feed == nil { // On board page. No feed.
		return
	}
	c.feed.GetBacklog <- backlogRequest{
		start:  ctr,
		client: c,
	}
}

// Close all conections an goroutines asociated with the Client
func (c *Client) closeConnections(err error) error {
	// Close update feed, if any
	if c.feed != nil {
		c.feed.Remove <- c
		c.feed = nil
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
		c.sendMessage(messageInvalid, err.Error())
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

// Sends a message to the client. Not safe for concurent use.
func (c *Client) send(msg []byte) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
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
	typ := messageType(uncast)
	if !c.synced && typ != messageSynchronise && typ != messageResynchronise {
		return errInvalidPayload(msg)
	}

	return c.runHandler(typ, msg)
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
	log.Printf("error by %s: %v\n", c.IP, err)
}

// Close closes a websocket connection with the provided status code and
// optional reason
func (c *Client) Close(err error) {
	select {
	case <-c.close:
	default:
		// Exit both for-select loops, if possible
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
