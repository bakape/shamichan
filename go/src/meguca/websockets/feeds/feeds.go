// Package feeds manages client synchronization to update feeds and provides a
// thread-safe interface for propagating messages to them and reassigning feeds
// to and from clients.
package feeds

import (
	"meguca/common"
	"sync"
)

// Contains and manages all active update feeds
var feeds = feedMap{
	// 64 len map to avoid some possible reallocation as the server starts
	feeds: make(map[uint64]*Feed, 64),
}

func init() {
	common.SendTo = SendTo
	common.ClosePost = ClosePost
}

// Container for managing client<->update-feed assignment and interaction
type feedMap struct {
	feeds map[uint64]*Feed
	mu    sync.RWMutex
}

// Add client to feed and send it the current status of the feed for
// synchronization to the feed's internal state
func addToFeed(id uint64, c common.Client) (feed *Feed, err error) {
	feeds.mu.Lock()
	defer feeds.mu.Unlock()

	feed, ok := feeds.feeds[id]
	if !ok {
		feed = &Feed{
			id:            id,
			add:           make(chan common.Client),
			remove:        make(chan common.Client),
			send:          make(chan []byte),
			insertPost:    make(chan postCreationMessage),
			insertImage:   make(chan postIDMessage),
			closePost:     make(chan postIDMessage),
			setOpenBody:   make(chan postBodyModMessage),
			clients:       make([]common.Client, 0, 8),
			messageBuffer: make([]byte, 0, 1<<10),
		}
		feeds.feeds[id] = feed
		err = feed.Start()
		if err != nil {
			return
		}
	}

	feed.add <- c
	return
}

// Remove client from a subscribed feed
func removeFromFeed(id uint64, c common.Client) {
	feeds.mu.Lock()
	defer feeds.mu.Unlock()

	feed := feeds.feeds[id]
	if feed == nil {
		return
	}
	feed.remove <- c
	// If the feeds sends a non-nil, it means it closed
	if nil != <-feed.remove {
		delete(feeds.feeds, feed.id)
	}
}

// SendTo sends a message to a feed, if it exists
func SendTo(id uint64, msg []byte) {
	feeds.mu.RLock()
	defer feeds.mu.RUnlock()

	feed := feeds.feeds[id]
	if feed != nil {
		feed.Send(msg)
	}
}

// ClosePost closes a post in a feed, if it exists
func ClosePost(id, op uint64, msg []byte) {
	feeds.mu.RLock()
	defer feeds.mu.RUnlock()

	feed := feeds.feeds[op]
	if feed != nil {
		feed.ClosePost(id, msg)
	}
}

// Remove all existing feeds and clients. Used only in tests.
func Clear() {
	feeds.mu.Lock()
	defer feeds.mu.Unlock()
	feeds.feeds = make(map[uint64]*Feed, 32)
}
