// Thread update feed managment

package websockets

import (
	"bytes"
	"log"
	"sync"

	"time"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
)

var (
	// Precompiled query for formatting the change feed stream
	formatChangeFeed = map[string]map[string]r.Term{
		"new_val": {
			"log": r.Row.
				Field("new_val").
				Field("log").
				Slice(r.Row.
					Field("old_val").
					Field("log").
					Count().
					Default(0),
				),
			"isFresh": r.Row.Field("old_val").Eq(nil),
		},
	}

	// Contains and manages all active update feeds
	feeds = feedContainer{
		// 100 len map to avoid some realocation as the server starts
		feeds: make(map[int64]*updateFeed, 100),
	}
)

// Container for holding and managing client<->update-feed interaction
type feedContainer struct {
	sync.RWMutex
	feeds map[int64]*updateFeed
}

// A feed with syncronisation logic of a certain thread
type updateFeed struct {
	id       int64                // Thread ID
	clients  []*Client            // Subscribed clients
	clientMu sync.RWMutex         // Protects the clients slice
	close    chan struct{}        // Close database change feed
	read     chan feedUpdate      // Read from database change feed
	buf      bytes.Buffer         // Buffer of unsent messages
	cache    map[int64]types.Post // Cache of posts updated within ght last 30 s
}

// Change feed update message
type feedUpdate struct {
	IsFresh bool
	types.Post
	Log [][]byte
}

// Add a client to an existing update feed or create a new one, if it does not
// exist yet
func (f *feedContainer) Add(id int64, cl *Client) error {
	f.Lock()
	defer f.Unlock()

	feed := f.feeds[id]
	if feed != nil {
		feed.add(cl)
		return nil
	}

	var err error
	feed, err = newUpdateFeed(id)
	if err != nil {
		close(feed.close)
		return err
	}

	f.feeds[id] = feed
	feed.add(cl)

	return nil
}

// Remove a client from the update feed
func (f *feedContainer) Remove(id int64, cl *Client) {
	f.RLock()
	feed, ok := f.feeds[id]
	f.RUnlock()

	if ok {
		feed.remove(cl)
	}
}

// Remove an updateFeed from f. Should only be called by the feed itself.
func (f *feedContainer) removeFeed(id int64) {
	f.Lock()
	defer f.Unlock()
	delete(f.feeds, id)
}

// Stop and remove all existing feeds. Used only in tests.
func (f *feedContainer) Clear() {
	f.Lock()
	defer f.Unlock()
	for id, feed := range f.feeds {
		select {
		case <-feed.close:
		default:
			close(feed.close)
		}
		delete(f.feeds, id)
	}
}

// Create a new updateFeed and sync it to the database
func newUpdateFeed(id int64) (feed *updateFeed, err error) {
	feed = &updateFeed{
		id:      id,
		clients: make([]*Client, 0, 1),
		close:   make(chan struct{}),
		read:    make(chan feedUpdate),
		cache:   make(map[int64]types.Post, 1<<4),
	}

	cursor, err := feed.streamUpdates()
	if err != nil {
		return
	}

	go feed.listen(cursor)

	return
}

// Start listening for updates from database and client requests
func (u *updateFeed) listen(cursor *r.Cursor) {
	send := time.NewTicker(time.Millisecond * 200)
	cleanCache := time.NewTicker(time.Second * 10)

	defer func() {
		send.Stop()
		cleanCache.Stop()
		feeds.removeFeed(u.id)

		err := cursor.Err()
		if err == nil {
			err = cursor.Close()
		}
		if err != nil {
			log.Printf("update feed: %s\n", err)
		}
	}()

	for {
		select {
		case update := <-u.read:
			if err := u.bufferUpdate(update); err != nil {
				log.Printf("feed update: %s", err)
				return
			}
		case <-send.C:
			u.flushBuffer()
		case t := <-cleanCache.C:
			u.cleanCache(t.Unix())
		case <-u.close: // Feed terminated externally
			return
		}
	}
}

// ConcatMessages concatenate multiple feed messages into a single one to reduce
// transport overhead
func ConcatMessages(msgs [][]byte) []byte {
	if len(msgs) == 1 {
		return msgs[0]
	}

	// Calculate capacity
	cap := 2 + len(msgs)
	for _, msg := range msgs {
		cap += len(msg)
	}

	buf := make([]byte, 2, cap)
	buf[0] = 52 // Corresponds to string encoded MessageConcat
	buf[1] = 50
	for i, msg := range msgs {
		if i != 0 {
			buf = append(buf, '\u0000') // Delimit with null bytes
		}
		buf = append(buf, msg...)
	}

	return buf
}

// StreamUpdates produces a stream of the replication log updates for the
// specified thread and sends it on read. Close the cursor to stop receiving
// updates. The intial contents of the log are assigned emediately.
func (u *updateFeed) streamUpdates() (*r.Cursor, error) {
	cursor, err := r.
		Table("posts").
		GetAllByIndex("op", u.id).
		Changes(r.ChangesOpts{
			IncludeInitial: true,
			Squash:         0.2, // Perform at most every 0.2 seconds
		}).
		Filter(r. // Exclude the initial swarm of old posts
				Row.
				Field("lastUpdated").
				Gt(r.Now().ToEpochTime().Sub(30)),
		).
		Merge(formatChangeFeed).
		Field("new_val").
		Without("ip", "password").
		Run(db.RSession)
	if err != nil {
		return nil, err
	}

	cursor.Listen(u.read)

	return cursor, nil
}

// Add client and send it posts updated within the last 30 seconds
func (u *updateFeed) add(c *Client) {
	u.clientMu.Lock()
	u.clients = append(u.clients, c)
	u.clientMu.Unlock()

	err := c.sendMessage(MessageSynchronise, u.cache)
	if err != nil {
		c.Close(err)
	}
}

// Remove client from listeners and close feed, if none left. Should only be
// called from "feeds".
func (u *updateFeed) remove(c *Client) {
	u.clientMu.Lock()
	defer u.clientMu.Unlock()

	if len(u.clients) == 1 {
		u.clients = []*Client{}
		select {
		case <-u.close:
		default:
			close(u.close)
		}
		return
	}

	for i, cl := range u.clients {
		if cl == c {
			copy(u.clients[i:], u.clients[i+1:])
			u.clients[len(u.clients)-1] = nil
			u.clients = u.clients[:len(u.clients)-1]
			break
		}
	}
}

// Buffer the replication log updates received from the DB and cache the new
// contents of the post.
func (u *updateFeed) bufferUpdate(update feedUpdate) error {
	u.cache[update.ID] = update.Post

	// To synchronise the client's state with u we resend any posts updated
	// withing the last 30 seconds. Client must dedup and render accordingly.
	if !update.IsFresh {
		data, err := EncodeMessage(MessageInsertPost, update.Post)
		if err != nil {
			return err
		}
		u.writeToBuffer(data)
	} else {
		for _, msg := range update.Log {
			u.writeToBuffer(msg)
		}
	}
	return nil
}

func (u *updateFeed) writeToBuffer(data []byte) {
	if u.buf.Len() != 0 {
		u.buf.WriteRune('\u0000')
	}
	u.buf.Write(data)
}

// Send any buffered messages
func (u *updateFeed) flushBuffer() {
	if u.buf.Len() == 0 {
		return
	}
	buf := prepependMessageType(MessageConcat, u.buf.Bytes())

	u.clientMu.RLock()
	for _, client := range u.clients {
		if err := client.send(buf); err != nil {
			client.Close(err)
		}
	}
	u.clientMu.RUnlock()

	u.buf.Reset()
}

// Clean up entries from updated post cache older than 30 seconds
func (u *updateFeed) cleanCache(time int64) {
	time -= 30
	for id, post := range u.cache {
		if post.LastUpdated < time {
			delete(u.cache, id)
		}
	}
}
