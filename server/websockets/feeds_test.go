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
	c.Assert(feed.log, DeepEquals, dummyLog)
	c.Assert(cap(feed.log), Equals, len(dummyLog)*2)

	// Add to exiting feed
	oldFeed := feed
	feed, err = feeds.Add(1, cl2)
	c.Assert(err, IsNil)
	c.Assert(feed, Equals, oldFeed)

	// Remove second client
	feed.Remove <- cl2
}

func (*DB) TestFeedLogAllocation(c *C) {
	feed := updateFeed{
		log: dummyLog,
	}
	added := []byte{1, 2, 3}
	feed.appendUpdate(added)
	c.Assert(feed.log, DeepEquals, append(dummyLog, added))
	c.Assert(cap(feed.log), Equals, 2*(len(dummyLog)+1))
}

func (*DB) TestStreamUpdates(c *C) {
	thread := types.DatabaseThread{
		ID:  1,
		Log: [][]byte{},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	// Empty log
	feed, err := newUpdateFeed(1)
	c.Assert(err, IsNil)
	c.Assert(feed.log, DeepEquals, [][]byte{})

	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	go cl.Listen()
	feed.Add <- cl

	log := [][]byte{[]byte("foo")}
	update := map[string][][]byte{"log": log}
	q := r.Table("threads").Get(1).Update(update)
	c.Assert(db.Write(q), IsNil)
	assertMessage(wcl, log[0], c)
	close(feed.close)
	cl.Close(nil)

	// Existing data
	feed, err = newUpdateFeed(1)
	c.Assert(err, IsNil)
	c.Assert(feed.log, DeepEquals, log)
	close(feed.close)
}
