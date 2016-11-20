package websockets

import (
	"bytes"
	"testing"
	"time"

	"strconv"

	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
	"github.com/bakape/meguca/common"
	r "github.com/dancannon/gorethink"
)

func TestAddingFeeds(t *testing.T) {
	assertTableClear(t, "posts")
	feeds.Clear()

	sv := newWSServer(t)
	defer sv.Close()
	sv.Add(2)
	cl1, wcl1 := sv.NewClient()
	go readListenErrors(t, cl1, sv)
	cl2, wcl2 := sv.NewClient()
	go readListenErrors(t, cl2, sv)

	feeds.Add <- subRequest{1, cl1}
	defer feeds.Clear()
	assertMessage(t, wcl1, "30{}")

	feeds.Add <- subRequest{1, cl2}
	assertMessage(t, wcl2, "30{}")

	feeds.Remove <- subRequest{1, cl2}

	cl1.Close(nil)
	cl2.Close(nil)
	sv.Wait()
}

func TestStreamUpdates(t *testing.T) {
	assertTableClear(t, "posts", "threads")
	feeds.Clear()
	assertInsert(t, "threads", common.DatabaseThread{
		ID:    1,
		Board: "a",
	})
	post := timestampedPost{
		Post: common.Post{
			ID: 1,
		},
		OP:          1,
		LastUpdated: time.Now().Unix(),
	}

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(1)
	go readListenErrors(t, cl, sv)
	feeds.Add <- subRequest{1, cl}
	defer feeds.Clear()

	assertMessage(t, wcl, "30{}")
	assertInsert(t, "posts", common.DatabasePost{
		StandalonePost: common.StandalonePost{
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
	sv.Add(1)
	go readListenErrors(t, cl2, sv)
	feeds.Add <- subRequest{1, cl2}
	std := encodeMessage(t, MessageSynchronise, map[int64]common.Post{
		1: post.Post,
	})
	assertMessage(t, wcl2, std)

	cl.Close(nil)
	cl2.Close(nil)
	sv.Wait()
}

func TestBufferUpdate(t *testing.T) {
	t.Parallel()

	stdPost := timestampedPost{
		Post: common.Post{
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
			name: "post inserted",
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
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			feeds := newFeedContainer()
			feeds.bufferUpdate(c.update)
			feed := feeds.feeds[c.update.OP]

			AssertDeepEquals(t, feed.cache[c.update.ID], c.cached)
			if s := feed.buf.String(); s != c.buf {
				LogUnexpected(t, c.buf, s)
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
		LogUnexpected(t, std, s)
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
	sv.Add(1)
	go readListenErrors(t, cl, sv)
	feeds := newFeedContainer()
	const msg = "a\u0000bc"
	feeds.feeds[1] = &updateFeed{
		clients:  []*Client{cl},
		buf:      *bytes.NewBufferString(msg),
		multiple: true,
	}

	feeds.flushBuffers()
	assertMessage(t, wcl, encodeMessageType(MessageConcat)+msg)

	cl.Close(nil)
	sv.Wait()
}

func encodeMessageType(typ MessageType) string {
	return strconv.Itoa(int(typ))
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
	AssertDeepEquals(t, feeds.feeds, std)
}
