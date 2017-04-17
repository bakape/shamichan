// Thread update feed management

package websockets

import (
	"bytes"
	"meguca/common"
	"meguca/db"
	"meguca/util"
	"strconv"
	"sync"
	"time"
)

// Contains and manages all active update feeds
var feeds = feedMap{
	// 32 len map to avoid some possible reallocation as the server starts
	feeds: make(map[uint64]*updateFeed, 32),
}

func init() {
	common.Feeds = &feeds
}

type postCreationMessage struct {
	hasImage bool
	id       uint64
	time     int64
	*bodyBuffer
	msg []byte
}

type postIDMessage struct {
	id  uint64
	msg []byte
}

type openPostCacheEntry struct {
	hasImage bool
	created  int64
	*bodyBuffer
}

// Container for managing client<->update-feed assignment and interaction
type feedMap struct {
	feeds map[uint64]*updateFeed
	mu    sync.RWMutex
}

// Add client and send it the current progress counter
func (f *feedMap) Add(id uint64, c *Client) (feed *updateFeed, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	feed, ok := f.feeds[id]
	if !ok {
		feed = &updateFeed{
			id:          id,
			add:         make(chan *Client),
			remove:      make(chan *Client),
			send:        make(chan []byte),
			insertPost:  make(chan postCreationMessage),
			insertImage: make(chan postIDMessage),
			closePost:   make(chan postIDMessage),
			clients:     make([]*Client, 0, 8),
		}
		f.feeds[id] = feed
		err = feed.Start()
		if err != nil {
			return
		}
	}

	feed.add <- c
	return
}

// Remove client from a subscribed feed
func (f *feedMap) Remove(feed *updateFeed, c *Client) {
	f.mu.Lock()
	defer f.mu.Unlock()

	feed.remove <- c
	// If the feeds sends a non-nil, it means it closed
	if nil != <-feed.remove {
		delete(f.feeds, feed.id)
	}
}

// SendTo sends a message to a feed, if it exists
func (f *feedMap) SendTo(id uint64, msg []byte) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	feed := f.feeds[id]
	if feed != nil {
		feed.Send(msg)
	}
}

// ClosePost closes a post in a feed, if it exists
func (f *feedMap) ClosePost(id, op uint64, msg []byte) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	feed := f.feeds[op]
	if feed != nil {
		feed.ClosePost(id, msg)
	}
}

// Remove all existing feeds and clients. Used only in tests.
func (f *feedMap) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.feeds = make(map[uint64]*updateFeed, 32)
}

// A feed with synchronization logic of a certain thread
type updateFeed struct {
	// Thread ID
	id uint64
	// Message flushing ticker
	ticker util.PausableTicker
	// Buffer of unsent messages
	util.MessageBuffer
	// Add a client
	add chan *Client
	// Remove client
	remove chan *Client
	// Propagates mesages to all listeners
	send chan []byte
	// Insert a new post into the thread and propagate to listeners
	insertPost chan postCreationMessage
	// Insert an image into an already allocated post
	insertImage chan postIDMessage
	// Close an open post
	closePost chan postIDMessage
	// Subscribed clients
	clients []*Client
	// Recent posts in the thread
	recent map[uint64]int64
	// Currently open posts
	open map[uint64]openPostCacheEntry
}

// Read existing posts into cache and start main loop
func (u *updateFeed) Start() (err error) {
	recent, err := db.GetRecentPosts(u.id)
	if err != nil {
		return
	}
	u.recent = make(map[uint64]int64, len(recent)*2)
	u.open = make(map[uint64]openPostCacheEntry, 16)
	for _, p := range recent {
		u.recent[p.ID] = p.Time
		u.open[p.ID] = openPostCacheEntry{
			hasImage: p.HasImage,
			created:  p.Time,
			bodyBuffer: &bodyBuffer{
				Buffer: *bytes.NewBuffer(p.Body),
			},
		}
	}

	go func() {
		// Stop the timer, if there are no messages and resume on new ones.
		// Keeping the goroutine asleep reduces CPU usage.
		u.ticker.Start()
		defer u.ticker.Pause()

		cleanUp := time.NewTicker(time.Minute)
		defer cleanUp.Stop()

		for {
			select {

			// Add client
			case c := <-u.add:
				u.clients = append(u.clients, c)
				c.Send(u.genSyncMessage())
				u.sendIPCount()

			// Remove client and close feed, if no clients left
			case c := <-u.remove:
				for i, cl := range u.clients {
					if cl == c {
						copy(u.clients[i:], u.clients[i+1:])
						u.clients[len(u.clients)-1] = nil
						u.clients = u.clients[:len(u.clients)-1]
						break
					}
				}
				if len(u.clients) != 0 {
					u.remove <- nil
					u.sendIPCount()
				} else {
					u.remove <- c
					return
				}

			// Buffer external message and prepare for sending to all clients
			case msg := <-u.send:
				u.bufferMessage(msg)

			// Send any buffered messages to any listening clients
			case <-u.ticker.C:
				buf := u.Flush()
				if buf == nil {
					u.ticker.Pause()
				} else {
					for _, c := range u.clients {
						c.Send(buf)
					}
				}

			// Remove stale cache entries (older than 15 minutes)
			case <-cleanUp.C:
				till := time.Now().Add(-15 * time.Minute).Unix()
				for id, created := range u.recent {
					if created < till {
						delete(u.recent, id)
					}
				}
				for id, p := range u.open {
					if p.created < till {
						delete(u.open, id)
					}
				}

			// Insert a new post, cache and propagate
			case p := <-u.insertPost:
				u.ticker.StartIfPaused()
				u.recent[p.id] = p.time
				u.open[p.id] = openPostCacheEntry{
					hasImage:   p.hasImage,
					created:    p.time,
					bodyBuffer: p.bodyBuffer,
				}
				u.Write(p.msg)

			// Insert and image into an open post
			case msg := <-u.insertImage:
				u.ticker.StartIfPaused()
				p := u.open[msg.id]
				p.hasImage = true
				u.open[msg.id] = p
				u.Write(msg.msg)

			// Close open post
			case msg := <-u.closePost:
				u.ticker.StartIfPaused()
				delete(u.open, msg.id)
				u.Write(msg.msg)
			}
		}
	}()

	return
}

// Send a message to all listening clients
func (u *updateFeed) Send(msg []byte) {
	u.send <- msg
}

// Buffer a message to be sent on the next tick
func (u *updateFeed) bufferMessage(msg []byte) {
	u.ticker.StartIfPaused()
	u.Write(msg)
}

// Generate a message for synchronizing to the current status of the update
// feed. The client has to compare this state to it's own and resolve any
// missing entries or conflicts.
func (u *updateFeed) genSyncMessage() []byte {
	b := make([]byte, 0, 1024)

	b = append(b, `30{"recent":[`...)
	first := true
	for id := range u.recent {
		if !first {
			b = append(b, ',')
		}
		first = false
		b = strconv.AppendUint(b, id, 10)
	}

	b = append(b, `],"open":{`...)

	first = true
	for id, p := range u.open {
		if !first {
			b = append(b, ',')
		}
		first = false

		b = append(b, '"')
		b = strconv.AppendUint(b, id, 10)
		b = append(b, `":{"hasImage":`...)

		b = strconv.AppendBool(b, p.hasImage)

		b = append(b, `,"body":`...)
		p.RLock()
		s := p.String()
		p.RUnlock()
		b = strconv.AppendQuote(b, s)

		b = append(b, '}')
	}

	b = append(b, `}}`...)

	return b
}

// Send unique IP count to all connected clients
func (u *updateFeed) sendIPCount() {
	ips := make(map[string]struct{}, len(u.clients))
	for _, c := range u.clients {
		ips[c.ip] = struct{}{}
	}

	msg, _ := common.EncodeMessage(common.MessageSyncCount, len(ips))
	u.bufferMessage(msg)
}

// Insert a new post into the thread and propagate to listeners
func (u *updateFeed) InsertPost(p *openPost, msg []byte) {
	u.insertPost <- postCreationMessage{
		hasImage:   p.hasImage,
		id:         p.id,
		time:       p.time,
		bodyBuffer: &p.bodyBuffer,
		msg:        msg,
	}
}

// Insert an image into an already allocated post
func (u *updateFeed) InsertImage(id uint64, msg []byte) {
	u.insertImage <- postIDMessage{
		id:  id,
		msg: msg,
	}
}

func (u *updateFeed) ClosePost(id uint64, msg []byte) {
	u.closePost <- postIDMessage{
		id:  id,
		msg: msg,
	}
}
