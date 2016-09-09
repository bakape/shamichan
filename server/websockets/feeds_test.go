package websockets

import (
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*DB) TestAddingFeeds(c *C) {
	writeSampleThread(c)
	cl1 := make(chan<- struct{})
	cl2 := make(chan<- struct{})

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
	feed.appendUpdates(added)
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
	read := make(chan []byte)
	closer := make(chan struct{})
	initial, err := streamUpdates(1, read, closer)
	c.Assert(err, IsNil)
	c.Assert(initial, DeepEquals, [][]byte{})

	log := []byte("foo")
	update := map[string][][]byte{"log": [][]byte{log}}
	q := r.Table("threads").Get(1).Update(update)
	c.Assert(db.Write(q), IsNil)
	c.Assert(<-read, DeepEquals, log)
	close(closer)

	// Existing data
	read = make(chan []byte)
	closer = make(chan struct{})
	initial, err = streamUpdates(1, read, closer)
	c.Assert(err, IsNil)
	c.Assert(initial, DeepEquals, [][]byte{log})
	close(closer)
}
