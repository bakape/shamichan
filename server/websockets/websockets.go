// Package websockets manages active websocket connections and messages received
// from and sent to them.
package websockets

import (
	"errors"
	"fmt"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	readTimeout  = time.Second * 30
	writeTimeout = time.Second * 10
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
	return u.Host == config.Config.HTTP.Origin
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
	synced       bool
	ident        auth.Ident
	subscription uint64
	ID           string
	conn         *websocket.Conn
	receiver     chan receivedMessage
	sender       chan []byte
	closer       chan struct{}
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

		// Without buffering, a busy client would block the subscription
		sender:   make(chan []byte, 1),
		receiver: make(chan receivedMessage),
		closer:   make(chan struct{}),
		conn:     conn,
	}
}

// Listen listens for incoming messages on the channels and processes them
func (c *Client) Listen() error {
	// Clean up, when loop exits
	defer Subs.Unlisten(c.subscription, c.ID)
	defer Clients.Remove(c.ID)

	go c.receiverLoop()
	for {
		select {
		case <-c.closer:
			return nil
		case msg := <-c.receiver:
			if msg.err != nil {
				return msg.err
			}
			if err := c.receive(msg.typ, msg.msg); err != nil {
				return err
			}
		case msg := <-c.sender:
			err := c.send(msg)
			if err != nil {
				return err
			}
		}
	}
}

// receiverLoop proxies the blocking conn.ReadMessage() into the main client
// select loop.
func (c *Client) receiverLoop() {
	// Handle websocket timeout
	c.conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})

	for {
		select {
		case <-c.closer:
			return
		default:
			typ, msg, err := c.conn.ReadMessage() // Blocking
			if err != nil {
				select {
				case <-c.closer:
				default:
					close(c.closer)
				}
				return
			}
			c.receiver <- receivedMessage{
				typ: typ,
				msg: msg,
				err: err,
			}
		}
	}
}

// receive parses a message received from the client through websockets
func (c *Client) receive(msgType int, msg []byte) error {
	if msgType != websocket.BinaryMessage {
		return c.Close(
			websocket.CloseUnsupportedData,
			"Only binary frames allowed",
		)
	}
	if c.ident.Banned {
		return c.Close(websocket.ClosePolicyViolation, "You are banned")
	}
	if len(msg) < 2 {
		return c.protocolError(msg)
	}

	typ := uint8(msg[0])
	if !c.synced && typ != messageSynchronise {
		return c.protocolError(msg)
	}

	data := msg[1:]
	switch typ {
	case messageInsertThread:
		// TODO: Actual handlers
		fmt.Println(data)
		return nil
	default:
		return c.protocolError(msg)
	}
}

// protocolError handles malformed messages received from the client
func (c *Client) protocolError(msg []byte) error {
	errMsg := fmt.Sprintf("Invalid message: %s", msg)
	if err := c.Close(websocket.CloseProtocolError, errMsg); err != nil {
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

// send sends a provided message as a websocket frame to the client
func (c *Client) send(msg []byte) error {
	return c.conn.WriteMessage(websocket.BinaryMessage, msg)
}

// Send thread-safely sends a message to the websocket client
func (c *Client) Send(msg []byte) {
	c.sender <- msg
}

// Close closes a websocket connection with the provided status code and
// optional reason
func (c *Client) Close(status int, reason string) error {
	select {
	case <-c.closer:
		return nil
	default:
		close(c.closer) // Stop any looping select statements listening to this
		return c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(status, reason),
			time.Now().Add(writeTimeout),
		)
	}
}
