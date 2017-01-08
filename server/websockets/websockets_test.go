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

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
	"github.com/gorilla/websocket"
)

const (
	invalidMessage   = "invalid message:"
	onlyText         = "only text frames allowed"
	abnormalClosure  = "websocket: close 1006"
	closeNormal      = "websocket: close 1000"
	invalidCharacter = "invalid character"
)

var (
	dialer = websocket.Dialer{}
)

type mockWSServer struct {
	t          testing.TB
	server     *httptest.Server
	connSender chan *websocket.Conn
	sync.WaitGroup
}

func init() {
	db.DBName = "meguca_test_websockets"
	db.IsTest = true
	if err := db.LoadDB(); err != nil {
		panic(err)
	}
	if err := Listen(); err != nil {
		panic(err)
	}
}

func newWSServer(t testing.TB) *mockWSServer {
	connSender := make(chan *websocket.Conn)
	handler := func(res http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(res, req, nil)
		if err != nil {
			t.Fatal(err)
		}
		connSender <- conn
	}
	return &mockWSServer{
		t:          t,
		connSender: connSender,
		server:     httptest.NewServer(http.HandlerFunc(handler)),
	}
}

func (m *mockWSServer) Close() {
	m.server.CloseClientConnections()
	m.server.Close()
	close(m.connSender)
}

func (m *mockWSServer) NewClient() (*Client, *websocket.Conn) {
	wcl := dialServer(m.t, m.server)
	return newClient(<-m.connSender, httptest.NewRequest("GET", "/", nil)), wcl
}

func dialServer(t testing.TB, sv *httptest.Server) *websocket.Conn {
	wcl, _, err := dialer.Dial(strings.Replace(sv.URL, "http", "ws", 1), nil)
	if err != nil {
		t.Fatal(err)
	}
	return wcl
}

func assertTableClear(t testing.TB, tables ...string) {
	if err := db.ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}

func assertInsert(t testing.TB, table string, doc interface{}) {
	if err := db.Insert(table, doc); err != nil {
		t.Fatal(err)
	}
}

func readListenErrors(t *testing.T, cl *Client, sv *mockWSServer) {
	defer sv.Done()
	if err := cl.listen(); err != nil {
		t.Fatal(err)
	}
}

func newRequest() *http.Request {
	return httptest.NewRequest("GET", "/", nil)
}

func assertMessage(t *testing.T, con *websocket.Conn, std string) {
	typ, msg, err := con.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if typ != websocket.TextMessage {
		t.Errorf("invalid received message format: %d", typ)
	}
	if s := string(msg); s != std {
		LogUnexpected(t, std, s)
	}
}

func assertWebsocketError(
	t *testing.T,
	conn *websocket.Conn,
	prefix string,
	sv *mockWSServer,
) {
	defer sv.Done()
	_, _, err := conn.ReadMessage()
	assertErrorPrefix(t, err, prefix)
}

func assertErrorPrefix(t *testing.T, err error, prefix string) {
	if errMsg := fmt.Sprint(err); !strings.HasPrefix(errMsg, prefix) {
		t.Fatalf("unexpected error prefix: `%s` : `%s`", prefix, errMsg)
	}
}

func TestLogError(t *testing.T) {
	const (
		ip  = "::1"
		msg = "Install Gentoo"
	)
	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.ip = ip

	log := captureLog(func() {
		cl.logError(errors.New(msg))
	})
	assertLog(t, log, fmt.Sprintf("error by %s: %s\n", ip, msg))
}

func captureLog(fn func()) string {
	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	fn()
	log.SetOutput(os.Stdout)
	return buf.String()
}

func assertLog(t *testing.T, input, std string) {
	std = `\d+/\d+/\d+ \d+:\d+:\d+ ` + std
	if strings.HasPrefix(std, input) {
		LogUnexpected(t, std, input)
	}
}

func TestTestClose(t *testing.T) {
	t.Parallel()

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	std := errors.New("foo")

	sv.Add(1)
	go func() {
		defer sv.Done()
		if err := cl.listen(); err != std {
			UnexpectedError(t, err)
		}
	}()
	cl.Close(std)
	sv.Wait()

	// Already closed
	cl.Close(nil)
}

func TestCloseMessageSending(t *testing.T) {
	t.Parallel()

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(2)

	go readListenErrors(t, cl, sv)
	go assertWebsocketError(t, wcl, closeNormal, sv)
	cl.Close(nil)
	sv.Wait()
}

func TestHandleMessage(t *testing.T) {
	t.Parallel()

	sv := newWSServer(t)
	defer sv.Close()
	msg := []byte("natsutte tsuchatta")

	// Non-text message
	cl, _ := sv.NewClient()
	err := cl.handleMessage(websocket.BinaryMessage, msg)
	assertErrorPrefix(t, err, onlyText)

	// Message too short
	msg = []byte("0")
	cl, _ = sv.NewClient()
	assertHandlerError(t, cl, msg, invalidMessage)

	// Unparsable message type
	msg = []byte("nope")
	assertHandlerError(t, cl, msg, invalidMessage)

	// Not a sync message, when not synced
	msg = []byte("99no")
	assertHandlerError(t, cl, msg, invalidMessage)

	// No handler
	cl.synced = true
	assertHandlerError(t, cl, msg, invalidMessage)

	// Invalid inner message payload. Test proper type reflection of the
	// errInvalidMessage error type
	msg = []byte("30nope")
	assertHandlerError(t, cl, msg, invalidCharacter)
}

func assertHandlerError(t *testing.T, cl *Client, msg []byte, prefix string) {
	err := cl.handleMessage(websocket.TextMessage, msg)
	assertErrorPrefix(t, err, prefix)
}

func TestInvalidMessage(t *testing.T) {
	t.Parallel()

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	sv.Add(1)
	go assertListenError(t, cl, onlyText, sv)
	if err := wcl.WriteMessage(websocket.BinaryMessage, []byte{1}); err != nil {
		t.Fatal(err)
	}
	assertMessage(t, wcl, `00"only text frames allowed"`)
	sv.Wait()
}

func assertListenError(
	t *testing.T,
	cl *Client,
	prefix string,
	sv *mockWSServer,
) {
	defer sv.Done()
	assertErrorPrefix(t, cl.listen(), prefix)
}

// Client properly closed connection with a control message
func TestClientClosure(t *testing.T) {
	t.Parallel()

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	sv.Add(1)
	go readListenErrors(t, cl, sv)
	normalCloseWebClient(t, wcl)
	sv.Wait()
}

func normalCloseWebClient(t *testing.T, wcl *websocket.Conn) {
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	deadline := time.Now().Add(time.Second)
	err := wcl.WriteControl(websocket.CloseMessage, msg, deadline)
	if err != nil {
		t.Error(err)
	}
}

func TestClientCleanUp(t *testing.T) {
	t.Parallel()

	sv := newWSServer(t)
	defer sv.Close()
	id := SyncID{
		OP:    1,
		Board: "a",
	}

	cl, wcl := sv.NewClient()
	Clients.add(cl, id)
	if _, sync := Clients.GetSync(cl); sync != id {
		LogUnexpected(t, id, sync)
	}

	sv.Add(1)
	go readListenErrors(t, cl, sv)
	normalCloseWebClient(t, wcl)
	sv.Wait()
	if synced, _ := Clients.GetSync(cl); synced {
		t.Error("client still synced")
	}
}

func TestHandler(t *testing.T) {
	t.Parallel()

	// Proper connection and client-side close
	sv := httptest.NewServer(http.HandlerFunc(Handler))
	defer sv.Close()
	wcl := dialServer(t, sv)
	normalCloseWebClient(t, wcl)
}

func TestSendMessage(t *testing.T) {
	t.Parallel()

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	cases := [...]struct {
		typ common.MessageType
		msg string
	}{
		{common.MessageInsertPost, "02null"},  // 1 char type string
		{common.MessageSynchronise, "30null"}, // 2 char type string
	}

	for i := range cases {
		c := cases[i]
		if err := cl.sendMessage(c.typ, nil); err != nil {
			t.Error(err)
		}
		assertMessage(t, wcl, c.msg)
	}
}

func TestPinging(t *testing.T) {
	old := pingTimer
	pingTimer = time.Millisecond
	defer func() {
		pingTimer = old
	}()

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	sv.Add(1)
	var once sync.Once
	wcl.SetPingHandler(func(_ string) error {
		once.Do(func() {
			sv.Done()
		})
		return nil
	})

	go wcl.ReadMessage()
	go cl.listen()
	sv.Wait()
	cl.Close(nil)
}
