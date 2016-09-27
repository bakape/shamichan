package websockets

import (
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*DB) TestOldFeedClosing(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()

	writeSampleThread(c)
	feed, err := feeds.Add(1, cl)
	c.Assert(err, IsNil)

	cl.feed = feed
	synchronise(nil, cl)
	c.Assert(cl.feed, IsNil)
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
	go readListenErrors(c, cl, sv)
	msg := syncRequest{
		Board:  "a",
		Thread: 1,
	}
	data := marshalJSON(msg, c)
	backlog1 := []byte("foog")
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

	res, err := EncodeMessage(MessageSynchronise, 2)
	c.Assert(err, IsNil)
	syncAssertMessage(wcl, res, c)

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
