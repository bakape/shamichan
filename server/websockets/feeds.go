// Thread update feed managment

package websockets

import (
	"log"
	"sync"

	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
)

var (
	// Precompiled query for extracting only the changed fields from the
	// replication log feed
	formatUpdateFeed = r.Branch(
		r.Row.HasFields("old_val"),
		r.Row.
			Field("new_val").
			Field("log").
			Slice(r.Row.Field("old_val").Field("log").Count()).
			Default(nil), // Thread deleted
		r.Row.Field("new_val").Field("log"),
	)

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
	id         int64               // Thread ID
	clients    []*Client           // Subscribed clients
	log        [][]byte            // Cached replication log
	Add        chan *Client        // Add a client to u
	Remove     chan *Client        // Remove a client from u
	GetBacklog chan backlogRequest // Request any missed messages
	close      chan struct{}       // Close database change feed
	read       chan [][]byte       // Read from database change feed
}

// A Client's request to receive missed data from the feed's replication log.
type backlogRequest struct {
	start  int // Index to slice at
	client *Client
}

// Add a client to an existing update feed or create a new one, if it does not
// exist yet
func (f *feedContainer) Add(id int64, cl *Client) (
	*updateFeed, error,
) {
	f.Lock()
	defer f.Unlock()

	feed := f.feeds[id]
	if feed != nil {
		feed.Add <- cl
		return feed, nil
	}

	var err error
	feed, err = newUpdateFeed(id)
	if err != nil {
		close(feed.close)
		return nil, err
	}

	f.feeds[id] = feed
	feed.Add <- cl

	return feed, nil
}

// Remove an updateFeed from f. Should only be called by the feed itself.
func (f *feedContainer) Remove(id int64) {
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
func newUpdateFeed(id int64) (*updateFeed, error) {
	feed := updateFeed{
		id:         id,
		clients:    make([]*Client, 0, 1),
		close:      make(chan struct{}),
		read:       make(chan [][]byte),
		Add:        make(chan *Client),
		Remove:     make(chan *Client),
		GetBacklog: make(chan backlogRequest),
	}

	cursor, err := feed.streamUpdates()
	if err != nil {
		return nil, err
	}

	go feed.Listen(cursor)

	return &feed, nil
}

// Start listening for updates from database and client requests
func (u *updateFeed) Listen(cursor *r.Cursor) {

	defer func() {
		err := cursor.Err()
		if err == nil {
			err = cursor.Close()
		}
		if err != nil {
			log.Printf("update feed: %s\n", err)
		}

		feeds.Remove(u.id)
	}()

	for {
		select {

		// Add client
		case client := <-u.Add:
			u.clients = append(u.clients, client)

		// Remove client or close feed, if no clients would remain
		case client := <-u.Remove:
			if len(u.clients) == 1 {
				u.clients = nil
				return
			}
			for i, cl := range u.clients {
				if cl == client {
					copy(u.clients[i:], u.clients[i+1:])
					u.clients[len(u.clients)-1] = nil
					u.clients = u.clients[:len(u.clients)-1]
					break
				}
			}

		// Update the log
		case msg := <-u.read:
			if msg == nil { // Thread deleted
				return
			}
			u.appendUpdate(msg)

		// Send the requested slice of the log to the client
		case req := <-u.GetBacklog:
			// Do nothing, if start == len(u.log) .
			// In rare racy cases start can also exceed u.log.
			if req.start < len(u.log) {
				msg := concatMessages(u.log[req.start:])
				if err := req.client.send(msg); err != nil {
					req.client.Close(err)
				}
			}

		// Feed terminated externally
		case <-u.close:
			return
		}
	}
}

// Append messagesto the replication log and send message to clients
func (u *updateFeed) appendUpdate(updates [][]byte) {
	// Create a new slice double the capacity, if capacity would be exceeded
	curLen := len(u.log)
	newLen := curLen + len(updates)
	if newLen > cap(u.log) {
		newLog := make([][]byte, curLen, newLen*2)
		copy(newLog, u.log)
		u.log = newLog
	}

	u.log = append(u.log, updates...)

	// Send update to all clients as a concatenated message
	concat := concatMessages(updates)
	for _, client := range u.clients {
		if err := client.send(concat); err != nil {
			client.Close(err)
		}
	}
}

// Concatenate multiple feed messages into a single one to reduce transport
// overhead
func concatMessages(msgs [][]byte) []byte {
	if len(msgs) == 1 {
		return msgs[0]
	}

	// Calculate capacity
	cap := 1
	for _, msg := range msgs {
		cap += len(msg) + 1
	}

	buf := make([]byte, 2, cap)
	buf[0] = 52
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
		Table("threads").
		Get(u.id).
		Changes(r.ChangesOpts{
			IncludeInitial: true,
			Squash:         1, // Perform at most every second
		}).
		Map(formatUpdateFeed).
		Run(db.RSession)
	if err != nil {
		return nil, err
	}

	var initial [][]byte
	if !cursor.Next(&initial) {
		if err := cursor.Err(); err != nil {
			return nil, err
		}
	}

	// Allocate twice as much as initial log length
	l := len(initial)
	u.log = make([][]byte, l, l*2)
	copy(u.log, initial)

	cursor.Listen(u.read)

	return cursor, nil
}
