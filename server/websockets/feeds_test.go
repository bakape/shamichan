package websockets

import (
	"bytes"
	"testing"
	"time"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
)

func TestAddingFeeds(t *testing.T) {
	assertTableClear(t, "posts")
	feeds.Clear()

	sv := newWSServer(t)
	defer sv.Close()
	cl1, wcl1 := sv.NewClient()
	cl2, wcl2 := sv.NewClient()

	feeds.Add <- subRequest{1, cl1}
	defer feeds.Clear()
	assertMessage(t, wcl1, "30{}")

	feeds.Add <- subRequest{1, cl2}
	assertMessage(t, wcl2, "30{}")

	feeds.Remove <- subRequest{1, cl2}
}

func TestStreamUpdates(t *testing.T) {
	assertTableClear(t, "posts", "threads")
	feeds.Clear()
	assertInsert(t, "threads", types.DatabaseThread{
		ID:    1,
		Board: "a",
	})
	post := timestampedPost{
		Post: types.Post{
			ID: 1,
		},
		OP:          1,
		LastUpdated: time.Now().Unix(),
	}

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	feeds.Add <- subRequest{1, cl}
	defer feeds.Clear()

	assertMessage(t, wcl, "30{}")
	assertInsert(t, "posts", types.DatabasePost{
		StandalonePost: types.StandalonePost{
			Post: post.Post,
			OP:   post.OP,
		},
		LastUpdated: post.LastUpdated,
		Log:         [][]byte{},
	})
	assertMessage(t, wcl, encodeMessage(t, MessageInsertPost, post.Post))

	q := db.FindPost(1).Update(map[string]interface{}{
		"log": appendLog([]byte("bar")),
	})
	if err := db.Write(q); err != nil {
		t.Fatal(err)
	}
	assertMessage(t, wcl, "bar")

	// Sending of cached posts
	cl2, wcl2 := sv.NewClient()
	feeds.Add <- subRequest{1, cl2}
	std := encodeMessage(t, MessageSynchronise, map[int64]types.Post{
		1: post.Post,
	})
	assertMessage(t, wcl2, std)
}

func TestBufferUpdate(t *testing.T) {
	t.Parallel()

	stdPost := timestampedPost{
		Post: types.Post{
			ID: 1,
		},
		OP: 1,
	}

	cases := [...]struct {
		name   string
		update feedUpdate
		cached timestampedPost
		buf    string
	}{
		{
			name: "post insertion",
			update: feedUpdate{
				Change:          postInserted,
				timestampedPost: stdPost,
				Log:             nil,
			},
			cached: stdPost,
			buf:    encodeMessage(t, MessageInsertPost, stdPost),
		},
		{
			name: "post updated",
			update: feedUpdate{
				Change:          postUpdated,
				timestampedPost: stdPost,
				Log:             [][]byte{[]byte("foo")},
			},
			cached: stdPost,
			buf:    "foo",
		},
		{
			name: "post deleted",
			update: feedUpdate{
				Change: postDeleted,
				timestampedPost: timestampedPost{
					Post: types.Post{
						ID: 1,
					},
				},
			},
			cached: timestampedPost{},
			buf:    encodeMessage(t, MessageDelete, 1),
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			feeds := newFeedContainer()
			if err := feeds.bufferUpdate(c.update); err != nil {
				t.Fatal(err)
			}
			feed := feeds.feeds[c.update.OP]

			assertDeepEquals(t, feed.cache[c.update.ID], c.cached)
			if s := feed.buf.String(); s != c.buf {
				logUnexpected(t, c.buf, s)
			}
		})
	}
}

func encodeMessage(t *testing.T, typ MessageType, data interface{}) string {
	msg, err := EncodeMessage(typ, data)
	if err != nil {
		t.Fatal(err)
	}
	return string(msg)
}

func TestWriteMultipleToBuffer(t *testing.T) {
	t.Parallel()

	u := updateFeed{}
	u.writeToBuffer([]byte("a"))
	u.writeToBuffer([]byte("b"))

	const std = "a\u0000b"
	if s := u.buf.String(); s != std {
		logUnexpected(t, std, s)
	}
	if !u.multiple {
		t.Fatal("containing multiple messages not recorded")
	}
}

func TestFlushMultipleMessages(t *testing.T) {
	t.Parallel()

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	feeds := newFeedContainer()
	const msg = "a\u0000bc"
	feeds.feeds[1] = &updateFeed{
		clients:  []*Client{cl},
		buf:      *bytes.NewBufferString(msg),
		multiple: true,
	}

	feeds.flushBuffers()
	assertMessage(t, wcl, `42`+msg)
}

func TestFeedCleanUp(t *testing.T) {
	t.Parallel()

	now := time.Now().Unix()
	expired := now - 31
	cls := []*Client{new(Client)}
	fresh := timestampedPost{
		LastUpdated: now,
	}
	stale := timestampedPost{
		LastUpdated: expired,
	}
	feeds := newFeedContainer()
	feeds.cursor = new(r.Cursor)
	feeds.feeds = map[int64]*updateFeed{
		1: {}, // No clients or cache
		2: { // No cache, has clients
			clients: cls,
		},
		3: { // Cache expired
			cache: map[int64]timestampedPost{
				1: stale,
			},
		},
		4: { // Not fully expired
			cache: map[int64]timestampedPost{
				1: stale,
				2: fresh,
			},
		},
	}

	feeds.cleanUp(now)

	std := map[int64]*updateFeed{
		2: {
			clients: cls,
		},
		4: {
			cache: map[int64]timestampedPost{
				2: fresh,
			},
		},
	}
	assertDeepEquals(t, feeds.feeds, std)
}
