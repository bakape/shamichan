// Thread update feed management

package websockets

import (
	"bytes"
	"log"
	"time"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/common"
	r "github.com/dancannon/gorethink"
)

// Post update kinds passed with feedUpdate
const (
	postInserted = iota
	postUpdated
)

var (
	// Contains and manages all active update feeds
	feeds = newFeedContainer()
)

// Container for holding and managing client<->update-feed interaction
type feedContainer struct {
	// Subscribe client
	Add chan subRequest
	// Remove client from subscribers
	Remove chan subRequest
	// Remove all existing feeds and clients. Used only in tests.
	clear chan struct{}
	// Read from "posts" table change feed
	read chan feedUpdate
	// Current database change feed cursor
	cursor *r.Cursor
	// Map of thread IDs to their feeds
	feeds map[int64]*updateFeed
}

// A feed with synchronization logic of a certain thread
type updateFeed struct {
	// Indicates the buf contains multiple concatenated messages
	multiple bool
	// Buffer of unsent messages
	buf bytes.Buffer
	// Subscribed clients
	clients []*Client
	// Cache of posts updated within the last 30 s
	cache map[int64]timestampedPost
}

// Change feed update message
type feedUpdate struct {
	Change uint8
	timestampedPost
	Log [][]byte
}

type timestampedPost struct {
	common.Post
	OP          int64 `json:"-"`
	LastUpdated int64 `json:"-"`
}

// Request to add or remove a client to a subscription
type subRequest struct {
	id     int64
	client *Client
}

// Listen initializes and starts listening for post updates from RethinkDB
func Listen() error {
	if err := feeds.streamUpdates(); err != nil {
		return err
	}
	go feeds.loop()
	return nil
}

// Separate function to ease testing
func newFeedContainer() feedContainer {
	return feedContainer{
		Add:    make(chan subRequest),
		Remove: make(chan subRequest),
		clear:  make(chan struct{}),
		read:   make(chan feedUpdate),

		// 100 len map to avoid some possible reallocation as the server starts
		feeds: make(map[int64]*updateFeed, 100),
	}
}

func (f *feedContainer) loop() {
	cleanUp := time.Tick(time.Second * 10)
	send := time.Tick(time.Millisecond * 200)

	for {
		select {
		case req := <-f.Add:
			f.addClient(req.id, req.client)
		case req := <-f.Remove:
			f.removeClient(req.id, req.client)
		case update := <-f.read:
			f.bufferUpdate(update)
		case <-f.clear:
			f.feeds = make(map[int64]*updateFeed, 1)
		case t := <-cleanUp:
			f.cleanUp(t.Unix())
		case <-send:
			f.flushBuffers()
		}
	}
}

// Add client and send it posts updated within the last 30 seconds
func (f *feedContainer) addClient(id int64, cl *Client) {
	feed, ok := f.feeds[id]
	if !ok {
		feed = &updateFeed{
			cache: make(map[int64]timestampedPost, 4),
		}
		f.feeds[id] = feed
	}
	feed.clients = append(feed.clients, cl)

	msg, err := EncodeMessage(MessageSynchronise, feed.cache)
	if err != nil {
		cl.Close(err)
	}
	cl.Send(msg)
}

// Remove client from subscribers
func (f *feedContainer) removeClient(id int64, cl *Client) {
	feed := f.feeds[id]
	for i, c := range feed.clients {
		if c == cl {
			copy(feed.clients[i:], feed.clients[i+1:])
			feed.clients[len(feed.clients)-1] = nil
			feed.clients = feed.clients[:len(feed.clients)-1]
			break
		}
	}
}

// Remove all existing feeds and clients. Used only in tests.
func (f *feedContainer) Clear() {
	f.clear <- struct{}{}
}

// Clean up entries from updated post cache older than 30 seconds. If a feed is
// to contain no listening clients or cached posts, remove it from the map.
// Also check if an error did not occur on the database feed.
func (f *feedContainer) cleanUp(time int64) {
	// If there's and error, log and attempt reconnecting
	if err := f.cursor.Err(); err != nil {
		log.Printf("update feed: %s\n", err)
		if err := f.streamUpdates(); err != nil { // Attempt to reconnect
			panic(err) // We're fucked
		}
		return
	}

	time -= 30

	for thread, feed := range f.feeds {
		for id, post := range feed.cache {
			if post.LastUpdated < time {
				delete(feed.cache, id)
			}
		}

		if len(feed.cache) == 0 && len(feed.clients) == 0 {
			delete(f.feeds, thread)
		}
	}
}

// Send any buffered messages to any listening clients
func (f *feedContainer) flushBuffers() {
	for _, feed := range f.feeds {
		if feed.buf.Len() == 0 {
			continue
		}
		if len(feed.clients) == 0 {
			feed.multiple = false
			feed.buf.Reset()
			continue
		}

		buf := feed.buf.Bytes()
		if feed.multiple {
			feed.multiple = false
			buf = prependMessageType(MessageConcat, buf)
		} else {
			// Need to copy, because the underlying array can be modified during
			// sending to clients.
			c := make([]byte, len(buf))
			copy(c, buf)
			buf = c
		}
		feed.buf.Reset()

		for _, client := range feed.clients {
			client.Send(buf)
		}
	}
}

// Subscribe to a stream of post updates and populate the initial cache of posts
// updated within the last 30 seconds.
func (f *feedContainer) streamUpdates() error {
	cursor, err := r.
		Table("posts").
		Between(r.Now().ToEpochTime().Sub(30), r.MaxVal, r.BetweenOpts{
			Index: "lastUpdated",
		}).
		Changes(r.ChangesOpts{
			IncludeInitial: true,
			IncludeTypes:   true,
			Squash:         0.2, // Perform at most every 0.2 seconds
		}).
		Map(func(ch r.Term) r.Term {
			return ch.Field("type").Do(func(typ r.Term) r.Term {
				return r.Branch(
					typ.Eq("add").Or(typ.Eq("initial")),
					ch.Field("new_val").Without("log", "ip", "password"),
					typ.Eq("remove"),
					nil,
					ch.Field("new_val").Merge(map[string]interface{}{
						"log": ch.
							Field("new_val").
							Field("log").
							Slice(ch.Field("old_val").Field("log").Count()),
						"change": postUpdated,
					}),
				)
			})
		}).
		Run(db.RSession)
	if err != nil {
		return err
	}

	cursor.Listen(f.read)
	f.cursor = cursor

	return nil
}

// Buffer the replication log updates received from the DB and cache the new
// contents of the post.
func (f *feedContainer) bufferUpdate(update feedUpdate) {
	// Empty updates are returned on post deletion
	if update.timestampedPost.ID == 0 {
		return
	}

	feed, ok := f.feeds[update.OP]
	if !ok {
		feed = &updateFeed{
			cache: make(map[int64]timestampedPost, 4),
		}
		f.feeds[update.OP] = feed
	}

	switch update.Change {
	// To synchronise the client's state with the feed we resend any posts
	// updated within the last 30 seconds. Client must deduplicate and render
	// accordingly.
	case postInserted:
		data, err := EncodeMessage(MessageInsertPost, update.Post)
		if err != nil {
			log.Printf("could not encode: %#v\n", update.Post)
			break
		}
		feed.writeToBuffer(data)
		feed.cache[update.ID] = update.timestampedPost
	// New replication log messages
	case postUpdated:
		for _, msg := range update.Log {
			feed.writeToBuffer(msg)
		}
		feed.cache[update.ID] = update.timestampedPost
	}
}

func (u *updateFeed) writeToBuffer(data []byte) {
	if u.buf.Len() != 0 {
		u.multiple = true
		u.buf.WriteRune('\u0000')
	}
	u.buf.Write(data)
}
