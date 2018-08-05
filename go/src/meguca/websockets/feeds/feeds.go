// Package feeds manages client synchronization to update feeds and provides a
// thread-safe interface for propagating messages to them and reassigning feeds
// to and from clients.
package feeds

import (
	"errors"
	"meguca/common"
	"sync"
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

// Export without circular dependency
func init() {
	common.SendTo = SendTo
	common.ClosePost = ClosePost
	common.BanPost = BanPost
	common.DeletePost = DeletePost
	common.DeleteImage = DeleteImage
	common.SpoilerImage = SpoilerImage
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
				id:              id,
				send:            make(chan []byte),
				insertPost:      make(chan postCreationMessage),
				closePost:       make(chan postCloseMessage),
				sendPostMessage: make(chan postMessage),
				setOpenBody:     make(chan postBodyModMessage),
				insertImage:     make(chan imageInsertionMessage),
				addWatcher:      make(chan *Watcher),
				removeWatcher:   make(chan *Watcher),
				messageBuffer:   make([]string, 0, 64),
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
		// If the feeds sends a non-nil, it means it closed
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
func ClosePost(
	id, op uint64,
	links []common.Link,
	commands []common.Command,
	msg []byte,
) {
	sendIfExists(op, func(f *Feed) {
		f.ClosePost(id, links, commands, msg)
	})
}

// BanPost propagates a message about a post being banned
func BanPost(id, op uint64) error {
	msg, err := common.EncodeMessage(common.MessageBanned, id)
	if err != nil {
		return err
	}

	return sendIfExists(op, func(f *Feed) {
		f.banPost(id, msg)
	})
}

// DeletePost propagates a message about a post being deleted
func DeletePost(id, op uint64) error {
	msg, err := common.EncodeMessage(common.MessageDeletePost, id)
	if err != nil {
		return err
	}
	return sendIfExists(op, func(f *Feed) {
		f.deletePost(id, msg)
	})
}

// DeleteImage propagates a message about an image being deleted from a post
func DeleteImage(id, op uint64) error {
	msg, err := common.EncodeMessage(common.MessageDeleteImage, id)
	if err != nil {
		return err
	}
	return sendIfExists(op, func(f *Feed) {
		f.DeleteImage(id, msg)
	})
}

// SpoilerImage propagates a message about an image being spoilered
func SpoilerImage(id, op uint64) error {
	msg, err := common.EncodeMessage(common.MessageSpoiler, id)
	if err != nil {
		return err
	}
	return sendIfExists(op, func(f *Feed) {
		f.SpoilerImage(id, msg)
	})
}

// Clear removes all existing feeds and clients. Used only in tests.
func Clear() {
	feeds.mu.Lock()
	defer feeds.mu.Unlock()
	feeds.feeds = make(map[uint64]*Feed, 32)
}
