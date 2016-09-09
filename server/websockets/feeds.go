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
			AtIndex(r.Row.Field("old_val").Field("log").Count()).
			Default(nil),
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
	id      int64                // Thread ID
	clients []chan<- struct{}    // Subscribed client update notification
	log     [][]byte             // Cached replication log
	Add     chan chan<- struct{} // Add a client to u
	Remove  chan chan<- struct{} // Remove a client from u
	Write   chan writeRequest    // Request to slice from the replication log
	close   chan struct{}        // Close database change feed
	read    chan []byte          // Read from database change feed
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
	read := make(chan []byte)
	initial, err := streamUpdates(id, read, cl)
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
		case msg := <-u.read:
			u.appendUpdates(msg)

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
func (u *updateFeed) appendUpdates(msg []byte) {
	// Create a new slice double the capacity, if capacity would be exceeded
	curLen := len(u.log)
	newLen := curLen + 1
	if newLen > cap(u.log) {
		newLog := make([][]byte, curLen, newLen*2)
		copy(newLog, u.log)
		u.log = newLog
	}

	u.log = append(u.log, msg)

	// Send to all clients that are not currently blocked doing some other
	// operation
	for _, client := range u.clients {
		select {
		case client <- struct{}{}:
		default:
		}
	}
}

// StreamUpdates produces a stream of the replication log updates for the
// specified thread and sends it on read. Close the close channel to stop
// receiving updates. The intial contents of the log are returned immediately.
func streamUpdates(id int64, write chan<- []byte, close <-chan struct{}) (
	[][]byte, error,
) {
	cursor, err := r.
		Table("threads").
		Get(id).
		Changes(r.ChangesOpts{IncludeInitial: true}).
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

	go func() {
		defer func() {
			err := cursor.Err()
			if err == nil {
				err = cursor.Close()
			}
			if err != nil {
				log.Printf("update feed: %s\n", err)
			}
		}()

		for {
			var msg []byte

			// Error or document deleted. Close cursor.
			if !cursor.Next(&msg) || msg == nil {
				break
			}

			select {
			case write <- msg:
			case <-close:
				return
			}
		}
	}()

	return initial, nil
}
