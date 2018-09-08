package feeds

import (
	"fmt"
	"time"

	"meguca/common"
	"meguca/db"

	"github.com/go-playground/log"
)

type postMessageType uint8

const (
	spoilerImage postMessageType = iota
	deletePost
	ban
	deleteImage
	meidoVision
	modLogPost
)

type postMessage struct {
	typ postMessageType
	id  uint64
	msg []byte
}

type postCreationMessage struct {
	common.Post
	msg []byte
}

type postCloseMessage struct {
	id       uint64
	links    []common.Link
	commands []common.Command
	msg      []byte
}

type imageInsertionMessage struct {
	id uint64
	common.Image
	msg []byte
}

type postBodyModMessage struct {
	id        uint64
	msg, body []byte
}

// Feed is a feed with synchronization logic of a certain thread
type Feed struct {
	// Thread ID
	id uint64
	// Message flushing ticker
	ticker
	// Common functionality
	baseFeed
	// Watchers currently subscibed to new closed post messages
	watchers      map[*Watcher]struct{}
	addWatcher    chan *Watcher
	removeWatcher chan *Watcher
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
	closePost chan postCloseMessage
	// Send various simple messages targeted at a specific post
	sendPostMessage chan postMessage
	// Set body of an open post
	setOpenBody chan postBodyModMessage
}

// Start read existing posts into cache and start main loop
func (f *Feed) Start() (err error) {
	thread, err := db.GetThread(f.id, 0)
	if err != nil {
		return
	}
	f.cache = newThreadCache(thread)

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
				if c.NewProtocol() {
					c.Send(f.cache.encodeThread(c.Last100()))
				} else {
					c.Send(f.cache.genSyncMessage())
				}
				f.sendIPCount()

			// Remove client and close feed, if no clients left
			case c := <-f.remove:
				if f.removeClient(c) {
					return
				}

				f.sendIPCount()

			case w := <-f.addWatcher:
				f.watchers[w] = struct{}{}
			case w := <-f.removeWatcher:
				delete(f.watchers, w)

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
			case p := <-f.insertPost:
				f.startIfPaused()
				f.cache.Posts[p.ID] = p.Post
				if p.msg != nil { // Post not being reclaimed by a DC-ed client
					f.write(p.msg)
					if f.cache.PostCtr <= 3000 {
						f.cache.BumpTime = time.Now().Unix()
					}
					f.cache.PostCtr++
					if p.Image != nil {
						f.cache.ImageCtr++
					}
				} else {
					f.cache.deleteMemoized(p.ID)
				}

			// Close an open post
			case msg := <-f.closePost:
				f.startIfPaused()

				p := f.cache.Posts[msg.id]
				p.Editing = false
				p.Links = msg.links
				p.Commands = msg.commands

				// Send partial closed post to thread watchers
				if len(f.watchers) != 0 {
					msg, err := encodeSSEMessage(f.id, p)
					if err != nil {
						log.Error(fmt.Errorf("SSE encoding: %s", err))
					}
					for w := range f.watchers {
						w.Send(msg)
					}
				}

				f.cache.Posts[msg.id] = p
				f.write(msg.msg)
				f.cache.deleteMemoized(msg.id)

			// Set the body of an open post and propagate
			case msg := <-f.setOpenBody:
				f.startIfPaused()
				p := f.cache.Posts[msg.id]
				p.Body = string(msg.body)
				f.cache.Posts[msg.id] = p
				f.write(msg.msg)
				f.cache.deleteMemoized(msg.id)

			case msg := <-f.insertImage:
				f.startIfPaused()
				p := f.cache.Posts[msg.id]
				p.Image = &msg.Image
				f.cache.Posts[msg.id] = p
				f.cache.ImageCtr++
				f.write(msg.msg)
				f.cache.deleteMemoized(msg.id)

			// Various post-related messages
			case msg := <-f.sendPostMessage:
				f.startIfPaused()
				switch msg.typ {
				case spoilerImage:
					p := f.cache.Posts[msg.id]
					if p.Image != nil {
						p.Image.Spoiler = true
					}
					f.cache.Posts[msg.id] = p
				case ban:
					p := f.cache.Posts[msg.id]
					p.Banned = true
					f.cache.Posts[msg.id] = p
				case deletePost:
					p := f.cache.Posts[msg.id]
					p.Deleted = true
					f.cache.Posts[msg.id] = p
				case deleteImage:
					p := f.cache.Posts[msg.id]
					p.Image = nil
					f.cache.Posts[msg.id] = p
				case meidoVision:
					p := f.cache.Posts[msg.id]
					p.MeidoVision = true
					f.cache.Posts[msg.id] = p
				}
				f.write(msg.msg)
				f.cache.deleteMemoized(msg.id)
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

	msg, _ := common.EncodeMessage(common.MessageSyncCount, struct {
		Active  int `json:"active"`
		Total   int `json:"total"`
	}{
		Active:  active,
		Total:   len(ips),
	})

	f.bufferMessage(msg)
}

// InsertPost inserts a new post into the thread or reclaim an open post after disconnect
// and propagate to listeners
func (f *Feed) InsertPost(post common.Post, msg []byte) {
	f.insertPost <- postCreationMessage{
		Post: post,
		msg:  msg,
	}
}

// InsertImage inserts an image into an already allocated post
func (f *Feed) InsertImage(id uint64, img common.Image, msg []byte) {
	f.insertImage <- imageInsertionMessage{
		id:    id,
		Image: img,
		msg:   msg,
	}
}

// Small helper method
func (f *Feed) _sendPostMessage(typ postMessageType, id uint64, msg []byte) {
	f.sendPostMessage <- postMessage{
		typ: typ,
		id:  id,
		msg: msg,
	}
}

// ClosePost closes a feed's post
func (f *Feed) ClosePost(
	id uint64,
	links []common.Link,
	commands []common.Command,
	msg []byte,
) {
	f.closePost <- postCloseMessage{id, links, commands, msg}
}

// SpoilerImage spoilers a feed's image
func (f *Feed) SpoilerImage(id uint64, msg []byte) {
	f._sendPostMessage(spoilerImage, id, msg)
}

func (f *Feed) banPost(id uint64, msg []byte) {
	f._sendPostMessage(ban, id, msg)
}

func (f *Feed) deletePost(id uint64, msg []byte) {
	f._sendPostMessage(deletePost, id, msg)
}

// DeleteImage deletes a feed's image
func (f *Feed) DeleteImage(id uint64, msg []byte) {
	f._sendPostMessage(deleteImage, id, msg)
}

func (f *Feed) meidoVision(id uint64, msg []byte) {
	f._sendPostMessage(meidoVision, id, msg)
}

func (f *Feed) modLogPost(id uint64, msg []byte) {
	f._sendPostMessage(modLogPost, id, msg)
}

// SetOpenBody sets the body of an open post and send update message to clients
func (f *Feed) SetOpenBody(id uint64, body, msg []byte) {
	f.setOpenBody <- postBodyModMessage{
		id:   id,
		msg:  msg,
		body: body,
	}
}
