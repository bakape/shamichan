package server

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	. "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

func (*ClientSuite) TestNewClient(c *C) {
	conn := &websocket.Conn{}
	req := newRequest(c)
	standard := Client{
		ident:    Ident{},
		Receiver: make(chan []byte),
		Sender:   make(chan []byte),
		Closer:   make(chan websocket.CloseError),
		conn:     conn,
	}
	client := NewClient(conn, req)
	c.Assert(client.id, Matches, "^[0-9a-zA-Z]{32}$")
	standard.id = client.id
	standard.Receiver = client.Receiver
	standard.Sender = client.Sender
	standard.Closer = client.Closer
	c.Assert(client, DeepEquals, &standard)
}

func (*ClientSuite) TestOpenClose(c *C) {
	client := newClient(c)
	c.Assert(client.isOpen(), Equals, true)
	client.setClosed()
	c.Assert(client.isOpen(), Equals, false)
}

func newClient(c *C) *Client {
	conn := &websocket.Conn{}
	req := newRequest(c)
	return NewClient(conn, req)
}

func (*ClientSuite) TestLogError(c *C) {
	const (
		ip  = "::1"
		msg = "Install Gentoo"
	)
	req := newRequest(c)
	req.RemoteAddr = ip
	cl := NewClient(&websocket.Conn{}, req)
	log := captureLog(func() {
		cl.logError(errors.New(msg))
	})
	assertLog(c, log, fmt.Sprintf("Error by %s: %s\n", ip, msg))
}

func (*ClientSuite) TestClose(c *C) {
	cl, sv, _, wcl := newConnectedClient(c)
	defer sv.CloseClientConnections()
	var wg sync.WaitGroup
	wg.Add(2)
	go readServerErrors(c, cl, &wg)
	go readClientErrors(c, wcl, &wg)
	c.Assert(normalCloseClient(cl), IsNil)
	wg.Wait()

	// Already closed
	c.Assert(normalCloseClient(cl), ErrorMatches, "^websocket: close sent$")
}

func readServerErrors(c *C, cl *Client, wg *sync.WaitGroup) {
	defer wg.Done()
	for cl.isOpen() {
		_, _, err := cl.conn.ReadMessage()
		if !cl.isOpen() {
			return
		}
		c.Assert(err, IsNil)
	}
}

func readClientErrors(c *C, conn *websocket.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		_, _, err := conn.ReadMessage()
		c.Assert(
			websocket.IsCloseError(err, websocket.CloseNormalClosure),
			Equals,
			true,
			Commentf("Unexpected error type: %v", err),
		)
		if err != nil {
			break
		}
	}
}

func normalCloseClient(cl *Client) error {
	return cl.close(websocket.CloseNormalClosure, "")
}

var dialer = websocket.Dialer{}

func newConnectedClient(c *C) (
	cl *Client, // Server-side client struct
	sv *httptest.Server, // Websocket server
	res http.ResponseWriter, // Server response
	wcl *websocket.Conn, // Client-side connection
) {
	handler := func(rs http.ResponseWriter, rq *http.Request) {
		conn, err := upgrader.Upgrade(rs, rq, nil)
		c.Assert(err, IsNil)
		cl = NewClient(conn, rq)
		res = rs
	}
	sv = httptest.NewServer(http.HandlerFunc(handler))
	wcl, _, err := dialer.Dial(strings.Replace(sv.URL, "http", "ws", 1), nil)
	c.Assert(err, IsNil)
	return
}

func (*ClientSuite) TestProtocolError(c *C) {
	const (
		msg    = "JIBUN WOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO"
		errMsg = "Invalid message: " + msg
	)
	cl, sv, _, wcl := newConnectedClient(c)
	defer sv.CloseClientConnections()
	var wg sync.WaitGroup
	wg.Add(2)
	go readServerErrors(c, cl, &wg)
	go func() {
		defer wg.Done()
		for {
			_, _, err := wcl.ReadMessage()
			c.Assert(
				err.Error(),
				Equals,
				"websocket: close 1002 (protocol error): "+errMsg,
			)
			if err != nil {
				break
			}
		}
	}()
	buf := []byte(msg)
	c.Assert(cl.protocolError(buf), ErrorMatches, errMsg)
	wg.Wait()

	// Already closed
	c.Assert(
		cl.protocolError(buf),
		ErrorMatches,
		errMsg+": websocket: close sent",
	)
}
