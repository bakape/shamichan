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
	invalidMessage  = "Invalid message: .*"
	onlyText        = "Only text frames allowed.*"
	abnormalClosure = "websocket: close 1006 .*"
	closeNormal     = "websocket: close 1000 .*"
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
	assertLog(c, log, fmt.Sprintf("Error by %s: %s\n", ip, msg))
}

func (*ClientSuite) TestClose(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	sv.Add(1)
	go func() {
		defer sv.Done()
		c.Assert(cl.Listen(), IsNil)
	}()
	cl.Close()
	sv.Wait()

	// Already closed
	cl.Close()
}

func (*ClientSuite) TestCloseMessageSending(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(2)
	go readListenErrors(c, cl, sv)
	go assertWebsocketError(c, wcl, closeNormal, sv)
	cl.Close()
	sv.Wait()
}

func (*ClientSuite) TestInvalidPayloadError(c *C) {
	const msg = "JIBUN WOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO"
	err := errInvalidPayload(msg)
	c.Assert(err, ErrorMatches, "Invalid message: "+msg)
}

func (*ClientSuite) TestSend(c *C) {
	std := []byte("foo")
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	go cl.Listen()
	cl.write <- std
	assertMessage(wcl, std, c)
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
	msg = []byte("12")
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
	asserHandlerError(cl, msg, "Invalid message:.*", c)
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
	assertMessage(wcl, []byte(`00"Only text frames allowed"`), c)
	sv.Wait()
}

func assertListenError(cl *Client, pattern string, sv *mockWSServer, c *C) {
	defer sv.Done()
	c.Assert(cl.Listen(), ErrorMatches, pattern)
}

// Client closes socket without message or timed out
func (*ClientSuite) TestClientTimeout(c *C) {
	sv := newWSServer(c)
	defer sv.Close()

	cl, wcl := sv.NewClient()
	oldPing := pingTimer
	oldRead := readTimeout
	pingTimer = time.Second
	readTimeout = time.Second * 2
	defer func() {
		pingTimer = oldPing
		readTimeout = oldRead
	}()

	// Ignore incomming pings
	wcl.SetPingHandler(func(string) error {
		return nil
	})

	// Timeout may occur either server or client-side, so we just make sure it
	// exits with an error
	c.Assert(cl.Listen(), NotNil)
}

func (*ClientSuite) TestPingPong(c *C) {
	sv := newWSServer(c)
	defer sv.Close()

	cl, wcl := sv.NewClient()
	oldPing := pingTimer
	oldRead := readTimeout
	pingTimer = time.Second
	readTimeout = time.Second * 2
	defer func() {
		pingTimer = oldPing
		readTimeout = oldRead
	}()

	sv.Add(1)
	go readListenErrors(c, cl, sv)
	go wcl.ReadMessage()

	// If Client outlives this with no errors, ping/pong is working
	time.Sleep(time.Second * 3)
	cl.Close()
	sv.Wait()
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
	deadline := time.Now().Add(writeTimeout)
	c.Assert(wcl.WriteControl(websocket.CloseMessage, msg, deadline), IsNil)
}

func (*ClientSuite) TestMessageSending(c *C) {
	sv := newWSServer(c)
	defer sv.Close()

	// Send a message
	std := []byte{127, 0, 0, 1}
	cl, wcl := sv.NewClient()
	go cl.Listen()
	cl.write <- std
	assertMessage(wcl, std, c)
}

func (*ClientSuite) TestCleanUp(c *C) {
	sv := newWSServer(c)
	defer sv.Close()

	cl, wcl := sv.NewClient()
	Clients.Add(cl, "1")
	c.Assert(Clients.clients[cl], Equals, "1")
	sv.Add(1)
	go readListenErrors(c, cl, sv)
	normalCloseWebClient(c, wcl)
	sv.Wait()
	_, ok := Clients.clients[cl]
	c.Assert(ok, Equals, false)
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
