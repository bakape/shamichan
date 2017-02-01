// Thread update feed management

package websockets

import (
	"bytes"
	"encoding/hex"
	"log"
	"strconv"
	"sync"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
	"github.com/lib/pq"
)

// Post update kinds passed with feedUpdate
const (
	postInserted = iota
	postUpdated
)

var (
	// Contains and manages all active update feeds
	feeds = newFeedMap()
)

// Container for managing client<->update-feed assignment and interaction
type feedMap struct {
	feeds map[uint64]*updateFeed
	mu    sync.Mutex
}

// Separate function to ease testing
func newFeedMap() *feedMap {
	return &feedMap{
		// 32 len map to avoid some possible reallocation as the server starts
		feeds: make(map[uint64]*updateFeed, 32),
	}
}

// Add client and send it the current progress counter
func (f *feedMap) Add(id uint64, c *Client) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	feed, ok := f.feeds[id]
	if !ok {
		feed = &updateFeed{
			close:   make(chan struct{}),
			clients: make([]*Client, 0, 8),
		}
		f.feeds[id] = feed
		if err := feed.Start(id); err != nil {
			return err
		}
	}

	feed.Lock()
	defer feed.Unlock()
	feed.clients = append(feed.clients, c)
	msg, err := common.EncodeMessage(common.MessageSynchronise, feed.ctr)
	if err != nil {
		return err
	}
	c.Send(msg)

	return nil
}

// Remove client from a subscribed feed
func (f *feedMap) Remove(id uint64, c *Client) {
	f.mu.Lock()
	defer f.mu.Unlock()

	feed := f.feeds[id]
	feed.Lock()
	defer feed.Unlock()
	for i, cl := range feed.clients {
		if cl == c {
			copy(feed.clients[i:], feed.clients[i+1:])
			feed.clients[len(feed.clients)-1] = nil
			feed.clients = feed.clients[:len(feed.clients)-1]
			break
		}
	}

	if len(feed.clients) == 0 {
		close(feed.close)
		delete(f.feeds, id)
	}
}

// Remove all existing feeds and clients. Used only in tests.
func (f *feedMap) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, feed := range f.feeds {
		close(feed.close)
	}
	f.feeds = make(map[uint64]*updateFeed, 32)
}

// A feed with synchronization logic of a certain thread
type updateFeed struct {
	// Update progress counter
	ctr uint64
	// Message flushing ticker
	ticker util.PausableTicker
	// Protects the client array and update counter
	sync.Mutex
	// Buffer of unsent messages
	db.MessageBuffer
	// Update channel and controller
	listener *pq.Listener
	// For breaking the inner loop
	close chan struct{}
	// Subscribed clients
	clients []*Client
}

func (u *updateFeed) Start(id uint64) (err error) {
	u.listener, err = db.Listen("t:" + strconv.FormatUint(id, 10))
	if err != nil {
		return
	}
	// Technically there might be some desync between these two calls, but we
	// can be almost certain, that the feed will already be started, when the
	// message is committed. Meh.
	u.ctr, err = db.ThreadCounter(id)
	if err != nil {
		return
	}

	go func() {
		// Stop the timer, if there are no messages and resume on new ones.
		// Keeping the goroutine asleep reduces CPU usage.
		u.ticker.Start()
		defer u.ticker.Pause()

		for {
			select {
			case <-u.close:
				u.ticker.StartIfPaused()
				if err := u.listener.Close(); err != nil {
					log.Printf("feed closing: %s", err)
				}
				return
			case msg := <-u.listener.Notify:
				u.ticker.StartIfPaused()
				if msg != nil { // Disconnect happened. Shouganai.
					u.decodeMessage(msg.Extra)
				}
			case <-u.ticker.C:
				u.flushBuffer()
			}
		}
	}()

	return
}

func (u *updateFeed) decodeMessage(data string) {
	buf, err := hex.DecodeString(data)
	if err != nil {
		log.Printf("could not decode update: %s : %s", data, err)
		return
	}

	// Split concatenated message
	if len(buf) > 2 && buf[0] == '3' && buf[1] == '3' {
		for _, msg := range bytes.Split(buf[2:], []byte{0}) {
			u.Write(msg)
		}
		return
	}

	u.Write(buf)
}

// Send any buffered messages to any listening clients
func (u *updateFeed) flushBuffer() {
	// Need to copy, because the underlying array can be modified during sending
	// to clients.
	buf, flushed := u.Flush(true)
	if flushed == 0 {
		u.ticker.Pause()
		return
	}

	u.Lock()
	defer u.Unlock()
	u.ctr += flushed

	if len(u.clients) == 0 {
		return
	}
	for _, c := range u.clients {
		c.Send(buf)
	}
}
