// Thread update feed managment

package websockets

import (
	"sync"

	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
)

// Precompiled query for extracting only the changed fields from the replication
// log feed
var formatUpdateFeed = r.Row.
	Field("new_val").
	Field("log").
	Slice(r.Row.
		Field("old_val").
		Field("log").
		Count().
		Default(0))

var feeds = feedContainer{
	// 100 len map to avoid some realocation as the server starts
	feeds: make(map[int64]*updateFeed, 100),
}

// Container for holding and managing client<->update-feed interaction
type feedContainer struct {
	sync.RWMutex
	feeds map[int64]*updateFeed
}

// A feed with syncronisation logic of a certain thread
type updateFeed struct {
	id      int64                // Thread ID
	clients []chan<- struct{}    // Subscribed client update notification
	log     [][]byte             // Cached replication log
	Add     chan chan<- struct{} // Add a client to u
	Remove  chan chan<- struct{} // Remove a client from u
	Write   chan writeRequest    // Request to slice from the replication log
	close   chan struct{}        // Close database change feed
	read    chan [][]byte        // Read from database change feed
}

// A Client's request to receive new data from the updateFeed's replication log.
type writeRequest struct {
	start int // Index to slice at
	write chan<- [][]byte
}

// Add a client to an existing update feed or create a new one, if it does not
// exist yet
func (f *feedContainer) Add(id int64, update chan<- struct{}) (
	*updateFeed, error,
) {
	f.Lock()
	defer f.Unlock()

	feed := f.feeds[id]
	if feed != nil {
		feed.Add <- update
		return feed, nil
	}

	var err error
	feed, err = newUpdateFeed(id)
	if err != nil {
		return nil, err
	}

	f.feeds[id] = feed
	go feed.Listen()
	feed.Add <- update

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
	cl := make(chan struct{})
	read := make(chan [][]byte)
	initial, err := db.StreamUpdates(id, read, cl)
	if err != nil {
		close(cl)
		return nil, err
	}

	// Allocate twice as much as initial log length
	l := len(initial)
	log := make([][]byte, l, l*2)
	copy(log, initial)

	feed := updateFeed{
		id:      id,
		clients: make([]chan<- struct{}, 0, 1),
		log:     log,
		close:   cl,
		read:    read,
		Add:     make(chan chan<- struct{}),
		Remove:  make(chan chan<- struct{}),
		Write:   make(chan writeRequest),
	}

	return &feed, nil
}

// Start listening for updates from database and client requests
func (u *updateFeed) Listen() {
	defer func() {
		select {
		case <-u.close:
		default:
			close(u.close)
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
		case updates := <-u.read:
			u.appendUpdates(updates)

		// Send the requested slice of the log to the client
		case req := <-u.Write:
			req.write <- u.log[req.start:]

		// Feed terminated externally
		case <-u.close:
			return
		}
	}
}

// Append messages to the replication log and notify any ready clients
func (u *updateFeed) appendUpdates(updates [][]byte) {
	// Create a new slice double the capacity, if capacity would be exceeded
	curLen := len(u.log)
	newLen := curLen + len(updates)
	if newLen > cap(u.log) {
		newLog := make([][]byte, curLen, newLen*2)
		copy(newLog, u.log)
		u.log = newLog
	}

	u.log = append(u.log, updates...)

	// Send to all clients that are not currently blocked doing some other
	// operation
	for _, client := range u.clients {
		select {
		case client <- struct{}{}:
		default:
		}
	}
}
