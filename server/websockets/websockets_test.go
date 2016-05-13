package websockets

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/gorilla/websocket"
	. "gopkg.in/check.v1"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	invalidPayload = `websocket: close 1007 .*`
	policyError    = `websocket: close 1008 .*`
	invalidMessage = "Invalid message: .*"
	onlyText       = "*. Only text frames allowed"
)

func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

func newRequest(c *C) *http.Request {
	req, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, IsNil)
	return req
}

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
	close(m.connSender)
}

var dialer = websocket.Dialer{}

func (m *mockWSServer) NewClient() (*Client, *websocket.Conn) {
	wcl := dialServer(m.c, m.server)
	return newClient(<-m.connSender), wcl
}

func dialServer(c *C, sv *httptest.Server) *websocket.Conn {
	wcl, _, err := dialer.Dial(strings.Replace(sv.URL, "http", "ws", 1), nil)
	c.Assert(err, IsNil)
	return wcl
}

func assertMessage(con *websocket.Conn, msg []byte, sv *mockWSServer, c *C) {
	defer sv.Done()
	typ, msg, err := con.ReadMessage()
	c.Assert(err, IsNil)
	c.Assert(typ, Equals, websocket.TextMessage)
	c.Assert(msg, DeepEquals, msg)
}

func (*ClientSuite) TestNewClient(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	c.Assert(cl.ID, Equals, "")
	c.Assert(cl.synced, Equals, false)
	c.Assert(cl.ident, DeepEquals, auth.Ident{IP: wcl.LocalAddr().String()})
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
	go readWebsocketErrors(c, cl.conn, sv)
	go readWebsocketErrors(c, wcl, sv)
	closeClient(c, cl)
	sv.Wait()

	// Already closed
	closeClient(c, cl)
}

func readWebsocketErrors(c *C, conn *websocket.Conn, sv *mockWSServer) {
	defer sv.Done()
	_, _, err := conn.ReadMessage()
	c.Assert(
		websocket.IsCloseError(err, websocket.CloseNormalClosure),
		Equals,
		true,
		Commentf("Unexpected error type: %v", err),
	)
}

func closeClient(c *C, cl *Client) {
	c.Assert(cl.Close(websocket.CloseNormalClosure, ""), IsNil)
}

func (*ClientSuite) TestInvalidPayload(c *C) {
	const msg = "JIBUN WOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO"
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(2)
	go assertWebsocketError(c, cl.conn, invalidPayload, sv)
	go assertWebsocketError(c, wcl, invalidPayload, sv)
	buf := []byte(msg)
	c.Assert(cl.invalidPayload(buf), ErrorMatches, invalidMessage)
	sv.Wait()
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
	sv.Add(1)
	go cl.Listen()
	go assertMessage(wcl, std, sv, c)
	cl.Send <- std
	sv.Wait()
}

func (*ClientSuite) TestParseMessage(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	msg := []byte("natsutte tsuchatta")

	// Non-text message
	cl, wcl := sv.NewClient()
	sv.Add(1)
	go assertWebsocketError(c, wcl, onlyText, sv)
	c.Assert(cl.handleMessage(websocket.BinaryMessage, msg), IsNil)
	sv.Wait()

	// Banned
	cl, wcl = sv.NewClient()
	cl.ident.Banned = true
	sv.Add(1)
	go assertWebsocketError(c, wcl, policyError+"You are banned", sv)
	c.Assert(cl.handleMessage(websocket.TextMessage, msg), IsNil)
	sv.Wait()

	// Message too short
	msg = []byte("12")
	cl, wcl = sv.NewClient()
	sv.Add(1)
	go assertWebsocketError(c, wcl, invalidPayload+invalidMessage, sv)
	c.Assert(
		cl.handleMessage(websocket.TextMessage, msg),
		ErrorMatches,
		invalidMessage,
	)
	sv.Wait()

	// Unparsable message type
	msg = []byte("nope")
	cl, wcl = sv.NewClient()
	sv.Add(1)
	go assertWebsocketError(c, wcl, invalidPayload, sv)
	c.Assert(
		cl.handleMessage(websocket.TextMessage, msg),
		ErrorMatches,
		invalidMessage,
	)
	sv.Wait()

	// Not a sync message, when not synced
	msg = []byte("99no")
	cl, wcl = sv.NewClient()
	sv.Add(1)
	go assertWebsocketError(c, wcl, invalidPayload, sv)
	c.Assert(
		cl.handleMessage(websocket.TextMessage, msg),
		ErrorMatches,
		invalidMessage,
	)
	sv.Wait()

	// No handler
	cl, wcl = sv.NewClient()
	cl.synced = true
	sv.Add(1)
	go assertWebsocketError(c, wcl, invalidPayload, sv)
	c.Assert(
		cl.handleMessage(websocket.TextMessage, msg),
		ErrorMatches,
		invalidMessage,
	)
	sv.Wait()
}

func (*ClientSuite) TestReceiverLoop(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	std := receivedMessage{
		typ: websocket.BinaryMessage,
		msg: []byte("shoganai wa ne"),
		err: nil,
	}

	cl, wcl := sv.NewClient()
	sv.Add(1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer sv.Done()
		c.Assert(<-cl.receive, DeepEquals, std)
	}()
	go func() {
		defer wg.Done()
		cl.receiverLoop()
	}()
	c.Assert(wcl.WriteMessage(websocket.BinaryMessage, std.msg), IsNil)
	sv.Wait()
	closeClient(c, cl)
	wg.Wait()

	// Already closed
	cl.receiverLoop()
}

func (*ClientSuite) TestCheckOrigin(c *C) {
	conf := config.ServerConfigs{}
	conf.HTTP.Origin = "fubar.com"
	config.Set(conf)

	// No header
	req := newRequest(c)
	c.Assert(CheckOrigin(req), Equals, true)

	// Invalid URL
	req = newRequest(c)
	req.Header.Set("Origin", "111111")
	c.Assert(CheckOrigin(req), Equals, false)

	// Matching header
	req = newRequest(c)
	req.Header.Set("Origin", "http://fubar.com")
	c.Assert(CheckOrigin(req), Equals, true)

	// Non-matching
	req = newRequest(c)
	req.Header.Set("Origin", "http://fubar.ru")
	c.Assert(CheckOrigin(req), Equals, false)
}

func (*ClientSuite) TestListen(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	msg := []byte{1, 2, 3}

	// Receive invalid message
	cl, wcl := sv.NewClient()
	sv.Add(2)
	go readListenErrors(c, cl, sv)
	go assertWebsocketError(c, wcl, onlyText, sv)
	c.Assert(wcl.WriteMessage(websocket.BinaryMessage, msg), IsNil)
	sv.Wait()

	// Client closed socket without message or timed out
	cl, wcl = sv.NewClient()
	oldR := readTimeout
	oldW := writeTimeout
	readTimeout = time.Second
	writeTimeout = readTimeout
	sv.Add(1)
	go readListenErrors(c, cl, sv)
	c.Assert(wcl.Close(), IsNil)
	sv.Wait()
	readTimeout = oldR
	writeTimeout = oldW

	// Client properly closed connection with a control message
	cl, wcl = sv.NewClient()
	sv.Add(1)
	go readListenErrors(c, cl, sv)
	normalCloseWebClient(c, wcl)
	sv.Wait()

	// Protocol error
	cl, wcl = sv.NewClient()
	sv.Add(2)
	go func() {
		defer sv.Done()
		c.Assert(cl.Listen(), ErrorMatches, invalidMessage)
	}()
	go assertWebsocketError(c, wcl, invalidPayload, sv)
	c.Assert(wcl.WriteMessage(websocket.TextMessage, []byte{123, 4}), IsNil)
	sv.Wait()

	// Send a message
	std := []byte{127, 0, 0, 1}
	cl, wcl = sv.NewClient()
	sv.Add(1)
	go cl.Listen()
	go func() {
		defer sv.Done()
		typ, msg, err := wcl.ReadMessage()
		c.Assert(err, IsNil)
		c.Assert(typ, Equals, websocket.TextMessage)
		c.Assert(msg, DeepEquals, std)
	}()
	cl.Send <- std
	sv.Wait()
}

func (*ClientSuite) TestCleanUp(c *C) {
	sv := newWSServer(c)
	defer sv.Close()

	cl, wcl := sv.NewClient()
	Clients.Add(cl)
	c.Assert(Clients.Has(cl.ID), Equals, true)
	sv.Add(1)
	go func() {
		defer sv.Done()
		c.Assert(cl.Listen(), IsNil)
	}()
	normalCloseWebClient(c, wcl)
	sv.Wait()
	c.Assert(Clients.Has(cl.ID), Equals, false)
}

func normalCloseWebClient(c *C, wcl *websocket.Conn) {
	err := wcl.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(writeTimeout),
	)
	c.Assert(err, IsNil)
}

func readListenErrors(c *C, cl *Client, sv *mockWSServer) {
	defer sv.Done()
	c.Assert(cl.Listen(), IsNil)
}

func (*ClientSuite) TestHandler(c *C) {
	// Proper connection and client-side close
	sv := httptest.NewServer(http.HandlerFunc(Handler))
	defer sv.Close()
	wcl := dialServer(c, sv)
	normalCloseWebClient(c, wcl)
}
