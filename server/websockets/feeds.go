// Thread update feed management

package websockets

import (
	"bytes"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
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
	// Count of buffered messages
	buffered uint64
	// Update progress counter
	ctr uint64
	// Protects the client array and update counter
	sync.Mutex
	// Buffer of unsent messages
	buf bytes.Buffer
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
		flush := time.NewTicker(time.Millisecond * 200)
		defer flush.Stop()

		for {
			select {
			case <-u.close:
				if err := u.listener.Close(); err != nil {
					log.Printf("feed closing: %s", err)
				}
				return
			case msg := <-u.listener.Notify:
				if msg != nil { // Disconnect happened. Shouganai.
					u.writeToBuffer(msg.Extra)
				}
			case <-flush.C:
				u.flushBuffer()
			}
		}
	}()

	return
}

func (u *updateFeed) writeToBuffer(data string) {
	if u.buf.Len() != 0 {
		u.buf.WriteRune('\u0000')
	}
	u.buf.WriteString(data)
	u.buffered++
}

// Send any buffered messages to any listening clients
func (u *updateFeed) flushBuffer() {
	if u.buffered == 0 {
		return
	}
	u.Lock()
	defer u.Unlock()
	defer func() {
		u.buf.Reset()
		u.ctr += u.buffered
		u.buffered = 0
	}()

	if len(u.clients) == 0 {
		return
	}

	buf := u.buf.Bytes()
	if u.buffered != 1 {
		buf = common.PrependMessageType(common.MessageConcat, buf)
	} else {
		// Need to copy, because the underlying array can be modified during
		// sending to clients.
		c := make([]byte, len(buf))
		copy(c, buf)
		buf = c
	}

	for _, c := range u.clients {
		c.Send(buf)
	}
}
