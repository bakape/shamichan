package feeds

import (
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	"github.com/go-playground/log"
)

type message struct {
	id  uint64
	msg []byte
}

type postCreationMessage struct {
	post       db.OpenPostMeta
	moderation []common.ModerationEntry
	message
}

type imageInsertionMessage struct {
	spoilered bool
	message
}

type postBodyModMessage struct {
	message
	body string
}

type moderationMessage struct {
	message
	entry common.ModerationEntry
}

type syncCount struct {
	Active int `json:"active"`
	Total  int `json:"total"`
}

// Feed is a feed with synchronization logic of a certain thread
type Feed struct {
	// Thread ID
	id uint64
	// Message flushing ticker
	ticker
	// Common functionality
	baseFeed
	// Buffer of unsent messages
	messageBuffer
	// Entire thread cached into memory
	cache threadCache
	// Propagates mesages to all listeners
	send chan []byte
	// Insert a new post into the thread and propagate to listeners
	insertPost chan postCreationMessage
	// Insert an image into an already allocated post
	insertImage chan imageInsertionMessage
	// Send message to close a post along with parsed post data
	closePost chan message
	// Send message to spoiler image of a specific post
	spoilerImage chan message
	// Set body of an open post
	setOpenBody chan postBodyModMessage
	// Send message about post moderation
	moderatePost chan moderationMessage
	// Let sent sync counter
	lastSyncCount syncCount
}

// Start read existing posts into cache and start main loop
func (f *Feed) Start() (err error) {
	f.cache, err = newThreadCache(f.id)
	if err != nil {
		return
	}

	go func() {
		// Stop the timer, if there are no messages and resume on new ones.
		// Keeping the goroutine asleep reduces CPU usage.
		f.start()
		defer f.pause()

		for {
			select {

			// Add client
			case c := <-f.add:
				f.addClient(c)

				msg, err := f.cache.getSyncMessage()
				if err != nil {
					log.Errorf("sync message: %s", err)
				}
				c.Send(msg)

				f.sendIPCount()

			// Remove client and close feed, if no clients left
			case c := <-f.remove:
				if f.removeClient(c) {
					return
				}

				f.sendIPCount()

			// Buffer external message and prepare for sending to all clients
			case msg := <-f.send:
				f.bufferMessage(msg)

			// Send any buffered messages to any listening clients
			case <-f.C:
				if buf := f.flush(); buf == nil {
					f.pause()
				} else {
					f.sendToAll(buf)
				}

			// Insert a new post, cache and propagate
			case msg := <-f.insertPost:
				f.modifyPost(msg.message, func(p *db.OpenPostMeta) {
					*p = msg.post
				})
				f.cache.All[msg.id] = msg.post.Page
				// Post can be automatically deleted on insertion
				if len(msg.moderation) != 0 {
					f.cache.Moderation[msg.id] = msg.moderation
				}
				f.sendIPCount()

			// Set the body of an open post and propagate
			case msg := <-f.setOpenBody:
				f.modifyPost(msg.message, func(p *db.OpenPostMeta) {
					p.Body = msg.body
				})

			case msg := <-f.insertImage:
				f.modifyPost(msg.message, func(p *db.OpenPostMeta) {
					p.HasImage = true
					p.Spoilered = msg.spoilered
				})

			case msg := <-f.spoilerImage:
				f.modifyPost(msg, func(p *db.OpenPostMeta) {
					p.Spoilered = true
				})

			case msg := <-f.closePost:
				f.startIfPaused()
				delete(f.cache.Open, msg.id)
				f.write(msg.msg)
				f.cache.clearMemoized()

			// Posts being moderated
			case msg := <-f.moderatePost:
				f.modifyPost(msg.message, func(p *db.OpenPostMeta) {
					switch msg.entry.Type {
					case common.PurgePost:
						p.Body = ""
						fallthrough
					case common.DeleteImage:
						p.HasImage = false
						p.Spoilered = false
					case common.SpoilerImage:
						p.Spoilered = true
					}
				})
				f.cache.Moderation[msg.id] = append(
					f.cache.Moderation[msg.id],
					msg.entry,
				)
			}
		}
	}()

	return
}

func (f *Feed) modifyPost(msg message, fn func(*db.OpenPostMeta)) {
	f.startIfPaused()

	p := f.cache.Open[msg.id]
	fn(&p)
	f.cache.Open[msg.id] = p

	if msg.msg != nil {
		f.write(msg.msg)
	}
	f.cache.clearMemoized()
}

// Send a message to all listening clients
func (f *Feed) Send(msg []byte) {
	f.send <- msg
}

// Buffer a message to be sent on the next tick
func (f *Feed) bufferMessage(msg []byte) {
	f.startIfPaused()
	f.write(msg)
}

// Send unique IP count to all connected clients
func (f *Feed) sendIPCount() {
	var active int
	ips := make(map[string]struct{}, len(f.clients))
	pastHour := time.Now().Add(-time.Hour).Unix()

	for c := range f.clients {
		ip := c.IP()
		if _, ok := ips[ip]; !ok && c.LastTime() >= pastHour {
			active++
		}
		ips[ip] = struct{}{}
	}

	new := syncCount{
		Active: active,
		Total:  len(ips),
	}
	if new != f.lastSyncCount {
		f.lastSyncCount = new
		msg, _ := common.EncodeMessage(common.MessageSyncCount, new)
		f.bufferMessage(msg)
	}
}

// Inserts a new post into the thread or reclaims an open post after disconnect
// and propagates to listeners
func (f *Feed) InsertPost(id uint64, p db.OpenPostMeta, msg []byte) {
	f.insertPost <- postCreationMessage{
		message: message{
			id:  id,
			msg: msg,
		},
		post: p,
	}
}

// InsertImage inserts an image into an already allocated post
func (f *Feed) InsertImage(id uint64, spoilered bool, msg []byte) {
	f.insertImage <- imageInsertionMessage{
		message: message{
			id:  id,
			msg: msg,
		},
		spoilered: spoilered,
	}
}

// ClosePost closes a feed's post
func (f *Feed) ClosePost(id uint64, msg []byte) {
	f.closePost <- message{
		id:  id,
		msg: msg,
	}
}

// SpoilerImage spoilers a feed's image
func (f *Feed) SpoilerImage(id uint64, msg []byte) {
	f.spoilerImage <- message{id, msg}
}

func (f *Feed) _moderatePost(id uint64, msg []byte,
	entry common.ModerationEntry,
) {
	f.moderatePost <- moderationMessage{
		message: message{
			id:  id,
			msg: msg,
		},
		entry: entry,
	}
}

// SetOpenBody sets the body of an open post and sends update message to
// clients
func (f *Feed) SetOpenBody(id uint64, body string, msg []byte) {
	f.setOpenBody <- postBodyModMessage{
		message: message{
			id:  id,
			msg: msg,
		},
		body: body,
	}
}
