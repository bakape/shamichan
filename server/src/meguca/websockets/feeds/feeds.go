// Package feeds manages client synchronization to update feeds and provides a
// thread-safe interface for propagating messages to them and reassigning feeds
// to and from clients.
package feeds

import (
	"errors"
	"sync"

	"meguca/common"
)

// Contains and manages all active update feeds
var feeds = feedMap{
	// 64 len map to avoid some possible reallocation as the server starts
	feeds:   make(map[uint64]*Feed, 64),
	tvFeeds: make(map[string]*tvFeed, 64),
	watchers: watcherMap{
		waiting: make(map[uint64]map[*Watcher]struct{}, 128),
		bound:   make(map[uint64]map[*Watcher]struct{}, 128),
	},
}

// Export to avoid circular dependency
func init() {
	common.SendTo = SendTo
	common.ClosePost = ClosePost
	common.PropagateModeration = PropagateModeration
}

// Thread watchers
type watcherMap struct {
	// not bound to a thread and waiting for feed
	waiting map[uint64]map[*Watcher]struct{}
	// threads watchers are bound and actively listening to
	bound map[uint64]map[*Watcher]struct{}
}

// Container for managing client<->update-feed assignment and interaction
type feedMap struct {
	feeds    map[uint64]*Feed
	tvFeeds  map[string]*tvFeed
	watchers watcherMap
	mu       sync.RWMutex
}

// Add client to feed and send it the current status of the feed for
// synchronization to the feed's internal state
func addToFeed(id uint64, board string, c common.Client) (
	feed *Feed, err error,
) {
	feeds.mu.Lock()
	defer feeds.mu.Unlock()

	var ok bool

	if id != 0 {
		feed, ok = feeds.feeds[id]
		if !ok {
			feed = &Feed{
				id:            id,
				send:          make(chan []byte),
				insertPost:    make(chan postCreationMessage),
				closePost:     make(chan postCloseMessage),
				spoilerImage:  make(chan message),
				moderatePost:  make(chan moderationMessage),
				setOpenBody:   make(chan postBodyModMessage),
				insertImage:   make(chan imageInsertionMessage),
				addWatcher:    make(chan *Watcher),
				removeWatcher: make(chan *Watcher),
				messageBuffer: make([]string, 0, 64),
			}

			// Bind all waiting watchers for this feed
			feed.watchers = feeds.watchers.waiting[id]
			if feed.watchers == nil {
				feed.watchers = make(map[*Watcher]struct{})
			}
			// Copy map to avoid locking.
			feeds.watchers.bound[id] = make(map[*Watcher]struct{},
				len(feed.watchers))
			for w := range feed.watchers {
				feeds.watchers.bound[id][w] = struct{}{}
			}
			delete(feeds.watchers.waiting, id)

			feed.baseFeed.init()
			feeds.feeds[id] = feed
			err = feed.Start()
			if err != nil {
				return
			}
		}
		feed.add <- c
	}

	return
}

// Subscribe to new posts from threads
func watchThreads(w *Watcher, threads []uint64) {
	feeds.mu.Lock()
	defer feeds.mu.Unlock()

	for _, id := range threads {
		feed := feeds.feeds[id]
		if feed != nil {
			feed.addWatcher <- w
			feeds.watchers.bound[id][w] = struct{}{}
		} else {
			m := feeds.watchers.waiting[id]
			if m == nil {
				m = make(map[*Watcher]struct{})
				feeds.watchers.waiting[id] = m
			}
			m[w] = struct{}{}
		}
	}
}

// Unwatch all threads subscribed to by this watcher
func unwatchThreads(w *Watcher) {
	feeds.mu.Lock()
	defer feeds.mu.Unlock()

	for id, watchers := range feeds.watchers.waiting {
		for watcher := range watchers {
			if watcher == w {
				if len(watchers) == 1 {
					delete(feeds.watchers.waiting, id)
					goto next
				}
				delete(watchers, w)
			}
		}
	next:
	}

	// Don't delete bound map, if empty, to keep both maps non-nil on the
	// feedMap and the Feed side
	for id, watchers := range feeds.watchers.bound {
		for watcher := range watchers {
			if watcher == w {
				feeds.feeds[id].removeWatcher <- w
				delete(watchers, w)
			}
		}
	}
}

// SubscribeToMeguTV subscribes to random video stream.
// Clients are automatically unsubscribed, when leaving their current sync feed.
func SubscribeToMeguTV(c common.Client) (err error) {
	feeds.mu.Lock()
	defer feeds.mu.Unlock()

	sync, _, board := GetSync(c)
	if !sync {
		return errors.New("meguTV: not synced")
	}

	tvf, ok := feeds.tvFeeds[board]
	if !ok {
		tvf = &tvFeed{}
		tvf.init()
		feeds.tvFeeds[board] = tvf
		err = tvf.start(board)
		if err != nil {
			return
		}
	}
	tvf.add <- c
	return
}

// Remove client from a subscribed feed
func removeFromFeed(id uint64, board string, c common.Client) {
	feeds.mu.Lock()
	defer feeds.mu.Unlock()

	if feed := feeds.feeds[id]; feed != nil {
		feed.remove <- c
		// If the feed sends a non-nil, it means it closed
		if nil != <-feed.remove {
			delete(feeds.feeds, feed.id)

			// Move all bound watchers back into waiting
			w := feeds.watchers.bound[feed.id]
			if len(w) != 0 { // Drop map, if empty
				feeds.watchers.waiting[feed.id] = w
			}
			delete(feeds.watchers.bound, feed.id)
		}
	}

	if feed := feeds.tvFeeds[board]; feed != nil {
		feed.remove <- c
		if nil != <-feed.remove {
			delete(feeds.tvFeeds, feed.board)
		}
	}
}

// SendTo sends a message to a feed, if it exists
func SendTo(id uint64, msg []byte) {
	sendIfExists(id, func(f *Feed) {
		f.Send(msg)
	})
}

// Run a send function of a feed, if it exists
func sendIfExists(id uint64, fn func(*Feed)) error {
	feeds.mu.RLock()
	defer feeds.mu.RUnlock()

	if feed := feeds.feeds[id]; feed != nil {
		fn(feed)
	}
	return nil
}

// InsertPostInto inserts a post into a tread feed, if it exists. Only use for
// already closed posts.
func InsertPostInto(post common.StandalonePost, msg []byte) {
	sendIfExists(post.OP, func(f *Feed) {
		f.InsertPost(post.Post, msg)
	})
}

// ClosePost closes a post in a feed, if it exists
func ClosePost(id, op uint64, links []common.Link, commands []common.Command,
) (err error) {
	msg, err := common.EncodeMessage(common.MessageClosePost, struct {
		ID       uint64           `json:"id"`
		Links    []common.Link    `json:"links,omitempty"`
		Commands []common.Command `json:"commands,omitempty"`
	}{
		ID:       id,
		Links:    links,
		Commands: commands,
	})
	if err != nil {
		return
	}

	sendIfExists(op, func(f *Feed) {
		f.ClosePost(id, links, commands, msg)
	})

	return
}

func PropagateModeration(id, op uint64, entry common.ModerationEntry,
) (err error) {
	msg, err := common.EncodeMessage(common.MessageModeratePost, struct {
		ID uint64 `json:"id"`
		common.ModerationEntry
	}{id, entry})
	if err != nil {
		return
	}

	sendIfExists(op, func(f *Feed) {
		f._moderatePost(id, msg, entry)
	})

	return
}

// Clear removes all existing feeds and clients. Used only in tests.
func Clear() {
	feeds.mu.Lock()
	defer feeds.mu.Unlock()
	feeds.feeds = make(map[uint64]*Feed, 32)
}
