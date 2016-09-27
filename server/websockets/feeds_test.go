package websockets

import (
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*DB) TestAddingFeeds(c *C) {
	writeSampleThread(c)
	sv := newWSServer(c)
	defer sv.Close()
	cl1, _ := sv.NewClient()
	cl2, _ := sv.NewClient()

	// Create new feed
	feed, err := feeds.Add(1, cl1)
	c.Assert(err, IsNil)
	c.Assert(feed, Equals, feeds.feeds[1])

	// Add to exiting feed
	oldFeed := feed
	feed, err = feeds.Add(1, cl2)
	c.Assert(err, IsNil)
	c.Assert(feed, Equals, oldFeed)

	// Remove second client
	feed.Remove <- cl2
}

func (*DB) TestStreamUpdates(c *C) {
	thread := types.DatabaseThread{
		ID:  1,
		Log: [][]byte{[]byte("foo")},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	// Empty log
	feed, err := newUpdateFeed(1)
	c.Assert(err, IsNil)
	c.Assert(feed.ctr, Equals, uint64(1))

	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	go cl.Listen()
	feed.Add <- cl

	msg := []byte("bar")
	q := r.Table("threads").Get(1).Update(map[string]r.Term{
		"log": appendLog(msg),
	})
	c.Assert(db.Write(q), IsNil)
	assertMessage(wcl, []byte("301"), c)
	assertMessage(wcl, msg, c)
	c.Assert(feed.ctr, Equals, uint64(2))
	close(feed.close)
	cl.Close(nil)

	// Existing data
	feed, err = newUpdateFeed(1)
	c.Assert(err, IsNil)
	c.Assert(feed.ctr, Equals, uint64(2))
	close(feed.close)
}
