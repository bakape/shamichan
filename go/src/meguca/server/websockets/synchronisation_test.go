package websockets

import (
	"meguca/auth"
	"meguca/common"
	"meguca/db"
	"meguca/imager/assets"
	. "meguca/test"
	"strconv"
	"testing"

	"github.com/gorilla/websocket"
)

func TestOldFeedClosing(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	if err := feeds.Add(1, cl); err != nil {
		t.Fatal(err)
	}
	defer feeds.Clear()

	cl.feedID = 1
	cl.synchronise(nil)
	if cl.feedID != 0 {
		t.Fatal("old feed not cleared")
	}
}

func TestSyncToBoard(t *testing.T) {
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
	skipMessage(t, wcl)

	// Valid synchronization
	msg.Board = "a"
	if err := cl.synchronise(marshalJSON(t, msg)); err != nil {
		t.Fatal(err)
	}
	defer Clients.Clear()
	skipMessage(t, wcl)
	assertMessage(t, wcl, `300`)
}

func skipMessage(t *testing.T, con *websocket.Conn) {
	_, _, err := con.ReadMessage()
	if err != nil {
		t.Error(err)
	}
}

func TestRegisterSync(t *testing.T) {
	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	defer Clients.Clear()

	syncs := [...]SyncID{
		{1, "a"},
		{2, "a"},
	}

	// Both for new syncs and switching syncs
	for _, s := range syncs {
		cl.registerSync(s.Board, s.OP)
		assertSyncID(t, &Clients, cl, s)
	}
}

func TestInvalidThreadSync(t *testing.T) {
	assertTableClear(t, "boards")

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
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)
	for _, msg := range [...]string{"foo", "bar"} {
		if err := db.UpdateLog(1, []byte(msg)); err != nil {
			t.Fatal(err)
		}
	}

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(1)
	go readListenErrors(t, cl, sv)
	data := marshalJSON(t, syncRequest{
		Board:  "a",
		Thread: 1,
	})

	if err := cl.synchronise(data); err != nil {
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

	skipMessage(t, wcl)
	assertMessage(t, wcl, `351`)
	assertMessage(t, wcl, "302")

	cl.Close(nil)
	sv.Wait()
}

func TestReclaimPost(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	const pw = "123"
	hash, err := auth.BcryptHash(pw, 6)
	if err != nil {
		t.Fatal(err)
	}
	posts := [...]db.Post{
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					Editing: true,
					Image:   &assets.StdJPEG,
					ID:      2,
					Body:    "abc\ndef",
					Time:    3,
				},
				OP:    1,
				Board: "a",
			},
			Password: hash,
		},
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					Editing: false,
					ID:      3,
				},
				OP:    1,
				Board: "a",
			},
			Password: hash,
		},
	}
	for _, p := range posts {
		if err := db.WritePost(nil, p); err != nil {
			t.Fatal(err)
		}
	}

	cases := [...]struct {
		name     string
		id       uint64
		password string
		code     int
	}{
		{"no post", 99, "", 1},
		{"already closed", 3, "", 1},
		{"wrong password", 2, "aaaaaaaa", 1},
		{"valid", 2, pw, 0},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			sv := newWSServer(t)
			defer sv.Close()
			cl, wcl := sv.NewClient()
			req := reclaimRequest{
				ID:       c.id,
				Password: c.password,
			}
			if err := cl.reclaimPost(marshalJSON(t, req)); err != nil {
				t.Fatal(err)
			}

			assertMessage(t, wcl, `31`+strconv.Itoa(c.code))
		})
	}
}
