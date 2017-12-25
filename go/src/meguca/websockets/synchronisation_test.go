package websockets

import (
	"meguca/common"
	. "meguca/test"
	"meguca/websockets/feeds"
	"testing"

	"github.com/gorilla/websocket"
)

func TestOldFeedClosing(t *testing.T) {
	feeds.Clear()
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	registerClient(t, cl, 1, "a")

	err := cl.synchronise(marshalJSON(t, syncRequest{
		Thread: 0,
		Board:  "a",
	}))
	if err != nil {
		t.Fatal(err)
	}

	if cl.feed != nil {
		t.Fatal("old feed not cleared")
	}
}

func TestSyncToBoard(t *testing.T) {
	feeds.Clear()
	setBoardConfigs(t, false)

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	// Invalid board
	msg := syncRequest{
		Thread: 0,
		Board:  "c",
	}
	if err := cl.synchronise(marshalJSON(t, msg)); err != errInvalidBoard {
		UnexpectedError(t, err)
	}

	// Valid synchronization
	msg.Board = "a"
	if err := cl.synchronise(marshalJSON(t, msg)); err != nil {
		t.Fatal(err)
	}
	assertMessage(t, wcl, "30null")
}

func skipMessage(t *testing.T, con *websocket.Conn) {
	t.Helper()
	_, _, err := con.ReadMessage()
	if err != nil {
		t.Error(err)
	}
}

func TestRegisterSync(t *testing.T) {
	feeds.Clear()
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()

	syncs := [...]struct {
		id    uint64
		board string
	}{
		{1, "a"},
		{0, "a"},
	}

	// Both for new syncs and switching syncs
	for _, s := range syncs {
		if err := cl.registerSync(s.id, s.board); err != nil {
			t.Fatal(err)
		}
		assertSyncID(t, cl, s.id, s.board)
	}
}

func assertSyncID(t *testing.T, cl *Client, id uint64, board string) {
	t.Helper()

	synced, _id, _board := feeds.GetSync(cl)
	if !synced {
		t.Error("client not synced")
	}
	if id != _id {
		LogUnexpected(t, id, _id)
	}
	if board != _board {
		LogUnexpected(t, board, _board)
	}
}

func TestInvalidThreadSync(t *testing.T) {
	feeds.Clear()
	assertTableClear(t, "boards")
	writeSampleBoard(t)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()

	data := marshalJSON(t, syncRequest{
		Board:  "a",
		Thread: 1,
	})
	if err := cl.synchronise(data); err != errInvalidThread {
		UnexpectedError(t, err)
	}
}

func TestSyncToThread(t *testing.T) {
	feeds.Clear()
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(1)
	go readListenErrors(t, cl, sv)

	sendMessage(t, wcl, common.MessageSynchronise, syncRequest{
		Board:  "a",
		Thread: 1,
	})

	skipMessage(t, wcl)
	skipMessage(t, wcl)
	assertMessage(t, wcl, "33341")
	assertSyncID(t, cl, 1, "a")

	cl.Close(nil)
	sv.Wait()
}

func sendMessage(
	t *testing.T,
	conn *websocket.Conn,
	typ common.MessageType,
	data interface{},
) {
	t.Helper()

	err := conn.WriteMessage(websocket.TextMessage, encodeMessage(t, typ, data))
	if err != nil {
		t.Fatal(err)
	}
}

func encodeMessage(
	t *testing.T,
	typ common.MessageType,
	data interface{},
) []byte {
	t.Helper()

	msg, err := common.EncodeMessage(typ, data)
	if err != nil {
		t.Fatal(err)
	}
	return msg
}
