package server

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bakape/meguca/auth"
	"github.com/gorilla/websocket"
	. "gopkg.in/check.v1"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	protocolError  = `websocket: close 1002 \(protocol error\): `
	policyError    = `websocket: close 1008 \(policy violation\): `
	invalidMessage = "Invalid message: "
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
	c.Assert(cl.ident, DeepEquals, auth.Ident{IP: wcl.LocalAddr().String()})
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

func captureLog(fn func()) string {
	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	fn()
	log.SetOutput(os.Stdout)
	return buf.String()
}

func assertLog(c *C, input, standard string) {
	c.Assert(input, Matches, `\d+/\d+/\d+ \d+:\d+:\d+ `+standard)
}

func (*ClientSuite) TestClose(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(2)
	go readServerErrors(c, cl, sv)
	go readClientErrors(c, wcl, sv)
	closeClient(c, cl)
	sv.Wait()

	// Already closed
	c.Assert(
		func() {
			cl.close(websocket.CloseNormalClosure, "")
		},
		PanicMatches,
		"^close of closed channel",
	)
}

func readServerErrors(c *C, cl *Client, sv *mockWSServer) {
	defer sv.Done()
	for cl.isOpen() {
		_, _, err := cl.conn.ReadMessage()
		if !cl.isOpen() {
			return
		}
		c.Assert(err, IsNil)
	}
}

func readClientErrors(c *C, conn *websocket.Conn, sv *mockWSServer) {
	defer sv.Done()
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
	sync.WaitGroup
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
		errMsg = invalidMessage + msg
	)
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(2)
	go readServerErrors(c, cl, sv)
	go assertWebsocketError(c, wcl, protocolError+errMsg, sv)
	buf := []byte(msg)
	c.Assert(cl.protocolError(buf), ErrorMatches, errMsg)
	sv.Wait()

	// Already closed
	c.Assert(
		func() {
			cl.protocolError(buf)
		},
		PanicMatches,
		"^close of closed channel",
	)
}

func assertWebsocketError(
	c *C,
	conn *websocket.Conn,
	errMsg string,
	sv *mockWSServer,
) {
	defer sv.Done()
	_, _, err := conn.ReadMessage()
	c.Assert(err, ErrorMatches, errMsg)
}

func (*ClientSuite) TestSend(c *C) {
	std := []byte("foo")
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(2)
	go readServerErrors(c, cl, sv)
	go func() {
		defer sv.Done()
		typ, msg, err := wcl.ReadMessage()
		c.Assert(err, IsNil)
		c.Assert(typ, Equals, websocket.BinaryMessage)
		c.Assert(msg, DeepEquals, std)
	}()
	c.Assert(cl.send(std), IsNil)
	closeClient(c, cl)
	sv.Wait()
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

func (*ClientSuite) TestReceive(c *C) {
	const (
		invalidLength = invalidMessage + "\x01"
		notSync       = invalidMessage + "@\x01"
	)
	sv := newWSServer(c)
	defer sv.Close()

	// Banned
	cl, wcl := sv.NewClient()
	cl.ident.Banned = true
	sv.Add(2)
	go readServerErrors(c, cl, sv)
	go assertWebsocketError(c, wcl, policyError+"You are banned", sv)
	c.Assert(cl.receive([]byte("natsutte tsuchatta")), IsNil)
	sv.Wait()

	// Message too short
	msg := []byte{1}
	cl, wcl = sv.NewClient()
	sv.Add(2)
	go readServerErrors(c, cl, sv)
	go assertWebsocketError(c, wcl, protocolError+invalidLength, sv)
	c.Assert(cl.receive(msg).Error(), Equals, invalidLength)
	sv.Wait()

	// Not a sync message, when not synced
	msg = []byte{64, 1}
	cl, wcl = sv.NewClient()
	sv.Add(2)
	go readServerErrors(c, cl, sv)
	go assertWebsocketError(c, wcl, protocolError+notSync, sv)
	c.Assert(cl.receive(msg).Error(), Equals, notSync)
	sv.Wait()

	// No handler
	cl, wcl = sv.NewClient()
	cl.synced = true
	sv.Add(2)
	go readServerErrors(c, cl, sv)
	go assertWebsocketError(c, wcl, protocolError+notSync, sv)
	c.Assert(cl.receive(msg).Error(), Equals, notSync)
	sv.Wait()
}

func (*ClientSuite) TestReceiverLoop(c *C) {
	const normalClose = `websocket: close 1000 \(normal\)`
	sv := newWSServer(c)
	defer sv.Close()
	msg := []byte("shouganai wa ne")

	// NOTE: Can't test client being already closed after returning from
	// c.conn.ReadMessage(), because it's a race condition.

	// Client diconnected
	cl, wcl := sv.NewClient()
	sv.Add(2)
	go runReceiveLoop(cl, sv)
	go assertWebsocketError(c, wcl, normalClose, sv)
	c.Assert(
		wcl.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second*5),
		),
		IsNil,
	)
	sv.Wait()

	// Text frame type
	sv.Add(2)
	cl, wcl = sv.NewClient()
	go runReceiveLoop(cl, sv)
	go assertWebsocketError(c, wcl, normalClose, sv)
	c.Assert(wcl.WriteMessage(websocket.TextMessage, msg), IsNil)
	c.Assert(<-cl.closer, DeepEquals, textFrameReceived)
	closeClient(c, cl)
	sv.Wait()

	// Proper message
	sv.Add(2)
	cl, wcl = sv.NewClient()
	go runReceiveLoop(cl, sv)
	go assertWebsocketError(c, wcl, normalClose, sv)
	c.Assert(wcl.WriteMessage(websocket.BinaryMessage, msg), IsNil)
	c.Assert(<-cl.receiver, DeepEquals, msg)
	closeClient(c, cl)
	sv.Wait()
}

func runReceiveLoop(cl *Client, sv *mockWSServer) {
	defer sv.Done()
	cl.receiverLoop()
}

func (*ClientSuite) TestListen(c *C) {
	sv := newWSServer(c)
	defer sv.Close()

	// Receive close request
	cl, wcl := sv.NewClient()
	sv.Add(3)
	go readServerErrors(c, cl, sv)
	go assertWebsocketError(
		c,
		wcl,
		`websocket: close 1011 \(internal server error\)`,
		sv,
	)
	go func() {
		defer sv.Done()
		c.Assert(cl.listen(), IsNil)
	}()
	cl.Close(websocket.CloseInternalServerErr, "")
	sv.Wait()

	// Receive a message
	const (
		invalid       = invalidMessage + "@"
		invalidClient = protocolError + invalid
	)
	msg := []byte{64}
	cl, wcl = sv.NewClient()
	sv.Add(3)
	go readServerErrors(c, cl, sv)
	go assertWebsocketError(c, wcl, invalidClient, sv)
	go func() {
		defer sv.Done()
		c.Assert(cl.listen(), ErrorMatches, invalid)
	}()
	cl.receiver <- msg
	sv.Wait()

	// Send a message
	cl, wcl = sv.NewClient()
	sv.Add(3)
	go readServerErrors(c, cl, sv)
	go func() {
		defer sv.Done()
		_, msg, err := wcl.ReadMessage()
		c.Assert(err, IsNil)
		c.Assert(msg, DeepEquals, msg)
	}()
	go func() {
		defer sv.Done()
		cl.sender <- msg
		cl.Close(websocket.CloseNormalClosure, "")
	}()
	c.Assert(cl.listen(), IsNil)
	sv.Wait()

	// Closed client
	cl, wcl = sv.NewClient()
	sv.Add(3)
	go readServerErrors(c, cl, sv)
	go func() {
		defer sv.Done()
		_, _, err := wcl.ReadMessage()
		c.Assert(err, IsNil)
	}()
	go func() {
		defer sv.Done()
		cl.setClosed()
		cl.sender <- msg
	}()
	c.Assert(cl.listen(), IsNil)
	closeClient(c, cl)
	sv.Wait()
	c.Assert(cl.listen(), IsNil)
}
