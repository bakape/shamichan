// Package feeds manages client synchronization to update feeds and provides a
// thread-safe interface for propagating messages to them and reassigning feeds
// to and from clients.
package feeds

import (
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	"github.com/bakape/pg_util"
)

// TODO: Board update feeds.
// If we follow a single server architecture (at least for all websocket
// clients), board feeds can just be an aggregation of all current active thread
// feeds - events, open posts and moderation data.

// Contains and manages all active update feeds
var feeds = feedMap{
	// 64 len map to avoid some possible reallocation as the server starts
	feeds:   make(map[uint64]*Feed, 64),
	tvFeeds: make(map[string]*tvFeed, 64),
}

// Container for managing client<->update-feed assignment and interaction
type feedMap struct {
	feeds   map[uint64]*Feed
	tvFeeds map[string]*tvFeed
	mu      sync.RWMutex
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
				closePost:     make(chan message),
				spoilerImage:  make(chan message),
				moderatePost:  make(chan moderationMessage),
				setOpenBody:   make(chan postBodyModMessage),
				insertImage:   make(chan imageInsertionMessage),
				messageBuffer: make([]string, 0, 64),
			}

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
	sendIfExists(id, func(f *Feed) error {
		f.Send(msg)
		return nil
	})
}

// Run a send function of a feed, if it exists
func sendIfExists(id uint64, fn func(*Feed) error) error {
	feeds.mu.RLock()
	defer feeds.mu.RUnlock()

	if feed := feeds.feeds[id]; feed != nil {
		fn(feed)
	}
	return nil
}

// TODO: Listed to close events from DB
// TODO: Propagate links and commands
// TODO: Propagate backlinks
func handlePostClosure(msg string) (err error) {
	// TODO: Propagate to board listeners without any thread listeners

	board, ints, err := db.SplitBoardAndInts(msg, 2)
	if err != nil {
		return
	}
	op := uint64(ints[0])
	id := uint64(ints[1])

	// Query for change data and endcode here instead of inside sendIfExists().
	// This has the overhead of querying post closure, even when nobody is
	// listening, but the gain is not locking feeds.mu for the duration of the
	// DB query and endcode. Most of the time a massage will arrive, only when
	// somebody is listening.
	data, err := db.GetPostCloseData(id)
	if err != nil {
		return
	}
	closeMsg, err := common.EncodeMessage(common.MessageClosePost, struct {
		ID       uint64                 `json:"id"`
		Links    map[uint64]common.Link `json:"links"`
		Commands json.RawMessage        `json:"commands"`
	}{
		ID:       id,
		Links:    data.Links,
		Commands: data.Commands,
	})
	if err != nil {
		return
	}

	sendIfExists(op, func(f *Feed) (err error) {
		f.ClosePost(id, closeMsg)
		return nil
	})

	// Propagate backlinks to any listeners. Also encode here to reduce feed.mu
	// mutex contention.
	source, err := json.Marshal(struct {
		ID    uint64 `json:"id"`
		OP    uint64 `json:"op"`
		Board string `json:"board"`
	}{
		ID:    id,
		OP:    op,
		Board: board,
	})
	if err != nil {
		return
	}
	for targetID, targetData := range data.Links {
		var msg []byte
		msg, err = common.EncodeMessage(common.MessageBacklink, struct {
			ID     uint64 `json:"id"`
			Source json.RawMessage
		}{
			ID:     targetID,
			Source: json.RawMessage(source), // Reuse common encoded part
		})
		if err != nil {
			return
		}

		// TODO: Propagate to board listeners without any thread listeners

		sendIfExists(targetData.OP, func(f *Feed) error {
			f.ClosePost(targetID, msg)
			return nil
		})
	}

	return
}

// Initialize internal runtime
func Init() (err error) {
	// TODO: Clean up feeds on thread and board deletion

	for _, o := range [...]pg_util.ListenOpts{
		{
			Channel: "post.moderated",
			OnMsg:   handlePostModeration,
		},
		{
			Channel: "post.closed",
			OnMsg:   handlePostClosure,
		},
		{
			Channel: "clients.disconnect",
			OnMsg:   handleClientDisconnecting,
		},
	} {
		err = db.Listen(o)
		if err != nil {
			return
		}
	}
	return
}

// Separate function for unit tests
func handlePostModeration(msg string) (err error) {
	// TODO: Propagate to board listeners without any thread listeners

	_, arr, err := db.SplitBoardAndInts(msg, 3)
	if err != nil {
		return
	}
	op, post, logID := arr[0], arr[1], arr[2]

	// Query performed here to prevent feeds.mu contention
	payload, err := db.GetModLogEntry(uint64(logID))
	if err != nil {
		return
	}
	var entry common.ModerationEntry
	err = json.Unmarshal(payload, &entry)
	if err != nil {
		return
	}

	return sendIfExists(uint64(op), func(f *Feed) (err error) {
		f._moderatePost(
			uint64(post),
			common.PrependMessageType(common.MessageModeratePost, payload),
			entry,
		)
		return
	})
}

// Separate function for unit tests
func handleClientDisconnecting(msg string) (err error) {
	split := strings.Split(msg, ",")
	if len(split) != 2 {
		return db.ErrMsgParse(msg)
	}
	board := split[0]
	ip := net.ParseIP(split[1])
	if ip == nil {
		return db.ErrMsgParse(msg)
	}

	payload, err := common.EncodeMessage(
		common.MessageInvalid,
		common.ErrBanned.Error(),
	)
	if err != nil {
		return
	}

	var cls []common.Client
	if board == "all" {
		cls = GetByIP(ip)
	} else {
		cls = GetByIPAndBoard(ip, board)
	}
	for _, cl := range cls {
		cl.Send(payload)
		cl.Close(nil)
	}
	return
}

// Clear removes all existing feeds and clients. Used only in tests.
func Clear() {
	feeds.mu.Lock()
	defer feeds.mu.Unlock()
	feeds.feeds = make(map[uint64]*Feed, 32)
}
