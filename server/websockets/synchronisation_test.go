package websockets

import (
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	"github.com/gorilla/websocket"
	. "gopkg.in/check.v1"
)

func (*DB) TestOldFeedClosing(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()

	writeSampleThread(c)
	msg := syncRequest{
		Thread: 0,
		Board:  "a",
	}
	data := marshalJSON(msg, c)
	c.Assert(synchronise(data, cl), IsNil)

	synchronise(nil, cl)
	c.Assert(cl.cursor, IsNil)
	c.Assert(cl.readFeed, IsNil)
}

func writeSampleThread(c *C) {
	c.Assert(db.Write(r.Table("threads").Insert(sampleThread)), IsNil)
}

func (*DB) TestSyncToBoard(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	// Invalid message
	c.Assert(synchronise(nil, cl), ErrorMatches, "unexpected end of JSON input")

	// Invalid board
	msg := syncRequest{
		Thread: 0,
		Board:  "c",
	}
	data := marshalJSON(msg, c)
	c.Assert(synchronise(data, cl), Equals, errInvalidBoard)

	// Valid synchronisation
	msg.Board = "a"
	data = marshalJSON(msg, c)
	c.Assert(synchronise(data, cl), IsNil)
	assertMessage(wcl, []byte(`300`), c)
}

func (*ClientSuite) TestRegisterSync(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	syncs := [...]SyncID{
		{1, "a"},
		{2, "a"},
	}

	// Both for new syncs and swicthing syncs
	for _, s := range syncs {
		registerSync(s.Board, s.OP, cl)
		synced, sync := Clients.GetSync(cl)
		c.Assert(synced, Equals, true)
		c.Assert(sync, Equals, s)
	}
}

func (*DB) TestInvalidThreadSync(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	msg := syncRequest{
		Board:  "a",
		Thread: 1,
	}
	data := marshalJSON(msg, c)
	c.Assert(synchronise(data, cl), Equals, errInvalidThread)
}

func (*DB) TestSyncToThread(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(1)

	msg := syncRequest{
		Board:  "a",
		Thread: 1,
	}
	data := marshalJSON(msg, c)
	backlog1 := []byte("foo")
	backlog2 := []byte("bar")
	thread := types.DatabaseThread{
		ID:    1,
		Board: "a",
		Log:   [][]byte{backlog1, backlog2},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	c.Assert(synchronise(data, cl), IsNil)
	_, sync := Clients.GetSync(cl)
	c.Assert(sync, Equals, SyncID{
		OP:    1,
		Board: "a",
	})
	go readListenErrors(c, cl, sv)

	assertSyncResponse(wcl, c)          // Receive client ID
	syncAssertMessage(wcl, backlog1, c) // Receive first missed message
	syncAssertMessage(wcl, backlog2, c) // Second message

	// Receive new message
	newMessage := []byte("foo")
	update := map[string]r.Term{
		"log": r.Row.Field("log").Append(newMessage),
	}
	c.Assert(db.Write(r.Table("threads").Get(1).Update(update)), IsNil)
	syncAssertMessage(wcl, newMessage, c)
	cl.Close(nil)
	sv.Wait()
}

func assertSyncResponse(wcl *websocket.Conn, c *C) {
	res, err := encodeMessage(messageSynchronise, 0)
	c.Assert(err, IsNil)
	syncAssertMessage(wcl, res, c)
}

// Test that only missed messages get sent as backlog.
func (*DB) TestOnlyMissedMessageSyncing(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	defer cl.Close(nil)
	sv.Add(1)
	go readListenErrors(c, cl, sv)

	msg := syncRequest{
		Board:  "a",
		Thread: 1,
		Ctr:    1,
	}
	data := marshalJSON(msg, c)
	backlogs := [][]byte{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	thread := types.DatabaseThread{
		ID:    1,
		Board: "a",
		Log:   backlogs,
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	c.Assert(synchronise(data, cl), IsNil)
	assertSyncResponse(wcl, c)             // Receive client ID
	syncAssertMessage(wcl, backlogs[1], c) // Receive first missed message
	syncAssertMessage(wcl, backlogs[2], c) // Second missed message

	cl.Close(nil)
	sv.Wait()
}
