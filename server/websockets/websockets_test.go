package websockets

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bakape/meguca/config"
	"github.com/gorilla/websocket"
	. "gopkg.in/check.v1"
)

const (
	invalidMessage   = "invalid message: .*"
	onlyText         = "only text frames allowed.*"
	abnormalClosure  = "websocket: close 1006 .*"
	closeNormal      = "websocket: close 1000 .*"
	invalidCharacter = "invalid character .*"
)

func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

func (*ClientSuite) SetUpTest(_ *C) {
	Clients.Clear()
	config.Set(config.Configs{}) // Reset configs on test start
}

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
	return newClient(<-m.connSender, httptest.NewRequest("GET", "/", nil)), wcl
}

func dialServer(c *C, sv *httptest.Server) *websocket.Conn {
	wcl, _, err := dialer.Dial(strings.Replace(sv.URL, "http", "ws", 1), nil)
	c.Assert(err, IsNil)
	return wcl
}

func assertMessage(con *websocket.Conn, std []byte, c *C) {
	typ, msg, err := con.ReadMessage()
	c.Assert(err, IsNil)
	c.Assert(typ, Equals, websocket.TextMessage)
	c.Assert(string(msg), Equals, string(std))
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

func (*ClientSuite) TestNewClient(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	c.Assert(cl.synced, Equals, false)
}

func (*ClientSuite) TestLogError(c *C) {
	const (
		ip  = "::1"
		msg = "Install Gentoo"
	)
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.IP = ip
	log := captureLog(func() {
		cl.logError(errors.New(msg))
	})
	assertLog(c, log, fmt.Sprintf("error by %s: %s\n", ip, msg))
}

func (*ClientSuite) TestClose(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	err := errors.New("foo")

	sv.Add(1)
	go func() {
		defer sv.Done()
		c.Assert(cl.Listen(), Equals, err)
	}()
	cl.Close(err)
	sv.Wait()

	// Already closed
	cl.Close(nil)
}

func (*ClientSuite) TestCloseMessageSending(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(2)
	go readListenErrors(c, cl, sv)
	go assertWebsocketError(c, wcl, closeNormal, sv)
	cl.Close(nil)
	sv.Wait()
}

func (*ClientSuite) TestInvalidPayloadError(c *C) {
	const msg = "JIBUN WOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO"
	err := errInvalidPayload(msg)
	c.Assert(err, ErrorMatches, "invalid message: "+msg)
}

func (*ClientSuite) TestHandleMessage(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	msg := []byte("natsutte tsuchatta")

	// Non-text message
	cl, _ := sv.NewClient()
	err := cl.handleMessage(websocket.BinaryMessage, msg)
	c.Assert(err, ErrorMatches, onlyText)

	// Message too short
	msg = []byte("0")
	cl, _ = sv.NewClient()
	asserHandlerError(cl, msg, invalidMessage, c)

	// Unparsable message type
	msg = []byte("nope")
	asserHandlerError(cl, msg, invalidMessage, c)

	// Not a sync message, when not synced
	msg = []byte("99no")
	asserHandlerError(cl, msg, invalidMessage, c)

	// No handler
	cl.synced = true
	asserHandlerError(cl, msg, invalidMessage, c)

	// Invalid inner message payload. Test proper type reflection of the
	// errInvalidMessage error type
	msg = []byte("30nope")
	asserHandlerError(cl, msg, invalidCharacter, c)
}

func asserHandlerError(cl *Client, msg []byte, pattern string, c *C) {
	err := cl.handleMessage(websocket.TextMessage, msg)
	c.Assert(err, ErrorMatches, pattern)
}

func (*ClientSuite) TestCheckOrigin(c *C) {
	config.AllowedOrigin = "fubar.com"

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

func (*ClientSuite) TestInvalidMessage(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	sv.Add(1)
	go assertListenError(cl, onlyText, sv, c)
	c.Assert(wcl.WriteMessage(websocket.BinaryMessage, []byte{1}), IsNil)
	assertMessage(wcl, []byte(`00"only text frames allowed"`), c)
	sv.Wait()
}

func assertListenError(cl *Client, pattern string, sv *mockWSServer, c *C) {
	defer sv.Done()
	c.Assert(cl.Listen(), ErrorMatches, pattern)
}

// Client properly closed connection with a control message
func (*ClientSuite) TestClientCleanClosure(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	sv.Add(1)
	go readListenErrors(c, cl, sv)
	normalCloseWebClient(c, wcl)
	sv.Wait()
}

func readListenErrors(c *C, cl *Client, sv *mockWSServer) {
	defer sv.Done()
	c.Assert(cl.Listen(), IsNil)
}

func normalCloseWebClient(c *C, wcl *websocket.Conn) {
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	deadline := time.Now().Add(time.Second)
	c.Assert(wcl.WriteControl(websocket.CloseMessage, msg, deadline), IsNil)
}

func (*ClientSuite) TestCleanUp(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	id := SyncID{
		OP:    1,
		Board: "a",
	}

	cl, wcl := sv.NewClient()
	Clients.Add(cl, id)
	_, sync := Clients.GetSync(cl)
	c.Assert(sync, Equals, id)
	sv.Add(1)
	go readListenErrors(c, cl, sv)
	normalCloseWebClient(c, wcl)
	sv.Wait()
	synced, _ := Clients.GetSync(cl)
	c.Assert(synced, Equals, false)
}

func (*ClientSuite) TestHandler(c *C) {
	// Proper connection and client-side close
	sv := httptest.NewServer(http.HandlerFunc(Handler))
	defer sv.Close()
	wcl := dialServer(c, sv)
	normalCloseWebClient(c, wcl)
}

func (*ClientSuite) TestSendMessage(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	// 1 char type string
	c.Assert(cl.sendMessage(messageInsertPost, nil), IsNil)
	assertMessage(wcl, []byte("02null"), c)

	// 2 char type string
	c.Assert(cl.sendMessage(messageSynchronise, nil), IsNil)
	assertMessage(wcl, []byte("30null"), c)
}
