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
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	c.Assert(cl.id, Matches, "^[0-9a-zA-Z]{32}$")
	c.Assert(cl.synced, Equals, false)
	c.Assert(cl.closed, Equals, false)
	c.Assert(cl.ident, DeepEquals, Ident{IP: wcl.LocalAddr().String()})
}

func (*ClientSuite) TestOpenClose(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	c.Assert(cl.isOpen(), Equals, true)
	cl.setClosed()
	c.Assert(cl.isOpen(), Equals, false)
}

func (*ClientSuite) TestLogError(c *C) {
	const (
		ip  = "::1"
		msg = "Install Gentoo"
	)
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.ident.IP = ip
	log := captureLog(func() {
		cl.logError(errors.New(msg))
	})
	assertLog(c, log, fmt.Sprintf("Error by %s: %s\n", ip, msg))
}

func (*ClientSuite) TestClose(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	var wg sync.WaitGroup
	wg.Add(2)
	go readServerErrors(c, cl, &wg)
	go readClientErrors(c, wcl, &wg)
	closeClient(c, cl)
	wg.Wait()

	// Already closed
	c.Assert(
		cl.close(websocket.CloseNormalClosure, ""),
		ErrorMatches,
		"^websocket: close sent$",
	)
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

func closeClient(c *C, cl *Client) {
	c.Assert(cl.close(websocket.CloseNormalClosure, ""), IsNil)
}

var dialer = websocket.Dialer{}

type mockWSServer struct {
	c          *C
	server     *httptest.Server
	connSender chan *websocket.Conn
}

func newWSServer(c *C) *mockWSServer {
	connSender := make(chan *websocket.Conn)
	handler := func(res http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(res, req, nil)
		c.Assert(err, IsNil)
		connSender <- conn
	}
	return &mockWSServer{
		c:          c,
		connSender: connSender,
		server:     httptest.NewServer(http.HandlerFunc(handler)),
	}
}

func (m *mockWSServer) Close() {
	m.server.CloseClientConnections()
	m.server.Close()
}

func (m *mockWSServer) NewClient() (*Client, *websocket.Conn) {
	wcl, _, err := dialer.Dial(
		strings.Replace(m.server.URL, "http", "ws", 1),
		nil,
	)
	m.c.Assert(err, IsNil)
	return NewClient(<-m.connSender), wcl
}

func (*ClientSuite) TestProtocolError(c *C) {
	const (
		msg    = "JIBUN WOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO"
		errMsg = "Invalid message: " + msg
	)
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	var wg sync.WaitGroup
	wg.Add(2)
	go readServerErrors(c, cl, &wg)
	go assertProtocolError(c, wcl, errMsg, &wg)
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

func assertProtocolError(
	c *C,
	conn *websocket.Conn,
	errMsg string,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	_, _, err := conn.ReadMessage()
	c.Assert(
		err.Error(),
		Equals,
		"websocket: close 1002 (protocol error): "+errMsg,
	)
}

func (*ClientSuite) TestSend(c *C) {
	std := []byte("foo")
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	var wg sync.WaitGroup
	wg.Add(2)
	go readServerErrors(c, cl, &wg)
	go func() {
		defer wg.Done()
		typ, msg, err := wcl.ReadMessage()
		c.Assert(err, IsNil)
		c.Assert(typ, Equals, websocket.BinaryMessage)
		c.Assert(msg, DeepEquals, std)
	}()
	c.Assert(cl.send(std), IsNil)
	closeClient(c, cl)
	wg.Wait()
}

func (*ClientSuite) TestExternalClose(c *C) {
	const (
		status = 2
		text   = "tsurupettan"
	)
	cl := Client{
		closer: make(chan websocket.CloseError),
	}
	go cl.Close(status, text)
	c.Assert(<-cl.closer, DeepEquals, websocket.CloseError{
		Code: status,
		Text: text,
	})
}

func (*ClientSuite) TestExternalSend(c *C) {
	std := []byte("WOW WOW")
	cl := Client{
		sender: make(chan []byte),
	}
	go cl.Send(std)
	c.Assert(<-cl.sender, DeepEquals, std)
}
