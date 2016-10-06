package websockets

import (
	"strings"
	"testing"
	"time"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
)

func TestOldFeedClosing(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", types.DatabasePost{
		Post: types.Post{
			ID: 1,
			OP: 1,
		},
	})

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	feeds.Add <- subRequest{1, cl}
	defer feeds.Clear()

	cl.feedID = 1
	synchronise(nil, cl)
	if cl.feedID != 0 {
		t.Fatal("old feed not cleared")
	}
}

func TestSyncToBoard(t *testing.T) {
	(*config.Get()).Boards = []string{"a"}

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	// Invalid board
	msg := syncRequest{
		Thread: 0,
		Board:  "c",
	}
	if err := synchronise(marshalJSON(t, msg), cl); err != errInvalidBoard {
		t.Errorf("unexpected error: %#v", err)
	}

	// Valid synchronisation
	msg.Board = "a"
	if err := synchronise(marshalJSON(t, msg), cl); err != nil {
		t.Fatal(err)
	}
	defer Clients.Clear()
	assertMessage(t, wcl, `30{}`)
}

func TestRegisterSync(t *testing.T) {
	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()

	syncs := [...]SyncID{
		{1, "a"},
		{2, "a"},
	}

	defer Clients.Clear()

	// Both for new syncs and swicthing syncs
	for _, s := range syncs {
		registerSync(s.Board, s.OP, cl)
		assertSyncID(t, &Clients, cl, s)
	}
}

func TestInvalidThreadSync(t *testing.T) {
	assertTableClear(t, "threads")

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()

	data := marshalJSON(t, syncRequest{
		Board:  "a",
		Thread: 1,
	})
	if err := synchronise(data, cl); err != errInvalidThread {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestSyncToThread(t *testing.T) {
	assertTableClear(t, "threads", "posts")
	assertInsert(t, "threads", types.DatabaseThread{
		ID:    1,
		Board: "a",
	})
	assertInsert(t, "posts", types.DatabasePost{
		Log: [][]byte{[]byte("foog"), []byte("bar")},
		Post: types.Post{
			ID:          1,
			OP:          1,
			Board:       "a",
			Body:        "foo",
			LastUpdated: time.Now().Unix(),
		},
	})

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(1)
	go readListenErrors(t, cl, sv)
	data := marshalJSON(t, syncRequest{
		Board:  "a",
		Thread: 1,
	})

	if err := synchronise(data, cl); err != nil {
		t.Fatal(err)
	}
	defer Clients.Clear()
	defer feeds.Clear()
	assertSyncID(t, &Clients, cl, SyncID{
		OP:    1,
		Board: "a",
	})
	if cl.feedID != 1 {
		t.Errorf("unexpected feed ID: 1 : %d", cl.feedID)
	}

	// The update stream does not guarantee initial message order on
	// synchronisation, only that messages from the same document will be in
	// order. Can't really test that here.
	_, msg, err := wcl.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if s := string(msg); !strings.HasPrefix(s, "30") {
		t.Fatalf("unexpected message type: %s", s)
	}

	cl.Close(nil)
	sv.Wait()
}
