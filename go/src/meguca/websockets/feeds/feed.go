package feeds

import (
	"meguca/common"
	"meguca/db"
	"strconv"
	"time"
)

type postMessageType uint8

const (
	closePost postMessageType = iota
	insertImage
	spoilerImage
	deletePost
	ban
	deleteImage
)

type postMessage struct {
	typ postMessageType
	id  uint64
	msg []byte
}

type postCreationMessage struct {
	open, hasImage bool
	id             uint64
	time           int64
	body, msg      []byte
}

type postBodyModMessage struct {
	id        uint64
	msg, body []byte
}

type openPostCacheEntry struct {
	hasImage, spoilered bool
	created             int64
	body                []byte
}

// A feed with synchronization logic of a certain thread
type Feed struct {
	// Thread ID
	id uint64
	// Message flushing ticker
	ticker
	// Buffer of unsent messages
	messageBuffer
	// Add a client
	add chan common.Client
	// Remove client
	remove chan common.Client
	// Propagates mesages to all listeners
	send chan []byte
	// Insert a new post into the thread and propagate to listeners
	insertPost chan postCreationMessage
	// Send various simple messages targeted at a specific post
	sendPostMessage chan postMessage
	// Set body of an open post
	setOpenBody chan postBodyModMessage
	// Subscribed clients
	clients []common.Client
	// Recent posts in the thread
	recent map[uint64]int64
	// Currently open posts
	open map[uint64]openPostCacheEntry
	// Deleted and banned posts
	deleted, deletedImage, banned []uint64
}

// Read existing posts into cache and start main loop
func (f *Feed) Start() (err error) {
	// Read recent post data into memory
	recent, err := db.GetRecentPosts(f.id)
	if err != nil {
		return
	}
	f.recent = make(map[uint64]int64, len(recent)*2)
	f.open = make(map[uint64]openPostCacheEntry, 16)
	for _, p := range recent {
		f.recent[p.ID] = p.Time
		f.open[p.ID] = openPostCacheEntry{
			hasImage:  p.HasImage,
			spoilered: p.Spoilered,
			created:   p.Time,
			body:      p.Body,
		}
	}

	// Read deleted and banned post IDs
	f.deleted, f.banned, err = db.GetThreadMutations(f.id)
	if err != nil {
		return
	}

	go func() {
		// Stop the timer, if there are no messages and resume on new ones.
		// Keeping the goroutine asleep reduces CPU usage.
		f.start()
		defer f.pause()

		cleanUp := time.NewTicker(time.Minute)
		defer cleanUp.Stop()

		for {
			select {

			// Add client
			case c := <-f.add:
				f.clients = append(f.clients, c)
				c.Send(f.genSyncMessage())
				f.sendIPCount()

			// Remove client and close feed, if no clients left
			case c := <-f.remove:
				for i, cl := range f.clients {
					if cl == c {
						copy(f.clients[i:], f.clients[i+1:])
						f.clients[len(f.clients)-1] = nil
						f.clients = f.clients[:len(f.clients)-1]
						break
					}
				}
				if len(f.clients) != 0 {
					f.remove <- nil
					f.sendIPCount()
				} else {
					f.remove <- c
					return
				}

			// Buffer external message and prepare for sending to all clients
			case msg := <-f.send:
				f.bufferMessage(msg)

			// Send any buffered messages to any listening clients
			case <-f.C:
				if buf := f.flush(); buf == nil {
					f.pause()
				} else {
					for _, c := range f.clients {
						c.Send(buf)
					}
				}

			// Remove stale cache entries (older than 15 minutes)
			case <-cleanUp.C:
				till := time.Now().Add(-15 * time.Minute).Unix()
				for id, created := range f.recent {
					if created < till {
						delete(f.recent, id)
					}
				}
				for id, p := range f.open {
					if p.created < till {
						delete(f.open, id)
					}
				}

			// Insert a new post, cache and propagate
			case p := <-f.insertPost:
				f.startIfPaused()
				f.recent[p.id] = p.time
				if p.open {
					f.open[p.id] = openPostCacheEntry{
						hasImage: p.hasImage,
						created:  p.time,
						body:     p.body,
					}
				}
				// Don't write insert messages, when reclaiming posts
				if p.msg != nil {
					f.write(p.msg)
				}

			// Set the body of an open post and propagate
			case msg := <-f.setOpenBody:
				f.startIfPaused()
				p := f.open[msg.id]
				p.body = msg.body
				f.open[msg.id] = p
				f.write(msg.msg)

			// Various post-related messages
			case msg := <-f.sendPostMessage:
				f.startIfPaused()
				switch msg.typ {
				case closePost:
					delete(f.open, msg.id)
				case insertImage:
					p := f.open[msg.id]
					p.hasImage = true
					f.open[msg.id] = p
				case spoilerImage:
					p := f.open[msg.id]
					p.spoilered = true
					f.open[msg.id] = p
				case ban:
					f.banned = append(f.banned, msg.id)
				case deletePost:
					f.deleted = append(f.deleted, msg.id)
				case deleteImage:
					f.deletedImage = append(f.deletedImage, msg.id)
				}
				f.write(msg.msg)
			}
		}
	}()

	return
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

// Generate a message for synchronizing to the current status of the update
// feed. The client has to compare this state to it's own and resolve any
// missing entries or conflicts.
// Handwritten to be as non-blocking as possible.
func (f *Feed) genSyncMessage() []byte {
	b := make([]byte, 0, 1<<10)

	first := true
	comma := func() {
		if !first {
			b = append(b, ',')
		}
		first = false
	}

	b = append(b, `30{"recent":[`...)
	for id := range f.recent {
		comma()
		b = strconv.AppendUint(b, id, 10)
	}

	b = append(b, `],"open":{`...)

	first = true
	for id, p := range f.open {
		comma()

		b = append(b, '"')
		b = strconv.AppendUint(b, id, 10)
		b = append(b, `":{"body":`...)
		b = strconv.AppendQuote(b, string(p.body))
		if p.hasImage {
			b = append(b, `,"hasImage":true`...)
		}
		if p.spoilered {
			b = append(b, `,"spoilered":true`...)
		}
		b = append(b, '}')
	}
	b = append(b, '}')

	encodeUints := func(key string, is []uint64) {
		b = append(b, `,"`...)
		b = append(b, key...)
		b = append(b, `":[`...)
		first = true
		for _, i := range is {
			comma()
			b = strconv.AppendUint(b, i, 10)
		}
		b = append(b, ']')
	}
	encodeUints("banned", f.banned)
	encodeUints("deleted", f.deleted)
	encodeUints("deletedImage", f.deletedImage)

	b = append(b, '}')

	return b
}

// Send unique IP count to all connected clients
func (f *Feed) sendIPCount() {
	ips := make(map[string]struct{}, len(f.clients))
	for _, c := range f.clients {
		ips[c.IP()] = struct{}{}
	}

	msg, _ := common.EncodeMessage(common.MessageSyncCount, len(ips))
	f.bufferMessage(msg)
}

// Insert a new post into the thread or reclaim an open post after disconnect
// and propagate to listeners
func (f *Feed) InsertPost(post common.StandalonePost, body, msg []byte) {
	f.insertPost <- postCreationMessage{
		open:     post.Editing,
		id:       post.ID,
		hasImage: post.Image != nil,
		time:     post.Time,
		body:     body,
		msg:      msg,
	}
}

// Insert an image into an already allocated post
func (f *Feed) InsertImage(id uint64, msg []byte) {
	f._sendPostMessage(insertImage, id, msg)
}

// Small helper method
func (f *Feed) _sendPostMessage(typ postMessageType, id uint64, msg []byte) {
	f.sendPostMessage <- postMessage{
		typ: typ,
		id:  id,
		msg: msg,
	}
}

func (f *Feed) ClosePost(id uint64, msg []byte) {
	f._sendPostMessage(closePost, id, msg)
}

func (f *Feed) SpoilerImage(id uint64, msg []byte) {
	f._sendPostMessage(spoilerImage, id, msg)
}

func (f *Feed) banPost(id uint64, msg []byte) {
	f._sendPostMessage(ban, id, msg)
}

func (f *Feed) deletePost(id uint64, msg []byte) {
	f._sendPostMessage(deletePost, id, msg)
}

func (f *Feed) deleteImage(id uint64, msg []byte) {
	f._sendPostMessage(deleteImage, id, msg)
}

// Set body of an open post and send update message to clients
func (f *Feed) SetOpenBody(id uint64, body, msg []byte) {
	f.setOpenBody <- postBodyModMessage{
		id:   id,
		msg:  msg,
		body: body,
	}
}
