package websockets

import . "gopkg.in/check.v1"

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
	feed.appendUpdates([][]byte{added})
	c.Assert(feed.log, DeepEquals, append(dummyLog, added))
	c.Assert(cap(feed.log), Equals, 2*(len(dummyLog)+1))
}
