package websockets

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/imager/assets"
	. "github.com/bakape/meguca/test"
)

func TestOldFeedClosing(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", common.DatabasePost{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				ID: 1,
			},
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
	setBoardConfigs(t, false)

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	// Invalid board
	msg := syncRequest{
		Thread: 0,
		Board:  "c",
	}
	if err := synchronise(marshalJSON(t, msg), cl); err != errInvalidBoard {
		UnexpectedError(t, err)
	}

	// Valid synchronization
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

	// Both for new syncs and switching syncs
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
		UnexpectedError(t, err)
	}
}

func TestSyncToThread(t *testing.T) {
	assertTableClear(t, "threads", "posts")
	assertInsert(t, "threads", common.DatabaseThread{
		ID:    1,
		Board: "a",
	})
	assertInsert(t, "posts", common.DatabasePost{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				ID:   1,
				Body: "foo",
			},
			OP:    1,
			Board: "a",
		},
		Log:         [][]byte{[]byte("foog"), []byte("bar")},
		LastUpdated: time.Now().Unix(),
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
	// synchronization, only that messages from the same document will be in
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

func TestReclaimPost(t *testing.T) {
	assertTableClear(t, "posts")

	const pw = "123"
	hash, err := auth.BcryptHash(pw, 6)
	if err != nil {
		t.Fatal(err)
	}
	assertInsert(t, "posts", []common.DatabasePost{
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					Editing: true,
					Image:   &assets.StdJPEG,
					ID:      1,
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
					ID:      2,
				},
			},
		},
	})

	cases := [...]struct {
		name     string
		id       uint64
		password string
		code     int
	}{
		{"no post", 99, "", 1},
		{"already closed", 2, "", 1},
		{"wrong password", 1, "aaaaaaaa", 1},
		{"valid", 1, pw, 0},
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
			reclaimPost(marshalJSON(t, req), cl)

			assertMessage(t, wcl, `31`+strconv.Itoa(c.code))
		})
	}
}
