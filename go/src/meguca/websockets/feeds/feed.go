package feeds

import (
	"meguca/common"
	"meguca/db"
	"time"
)

// TODO: Propagate thread modetation events to all clients live

type postMessageType uint8

const (
	closePost postMessageType = iota
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
	common.Post
	msg []byte
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

// A feed with synchronization logic of a certain thread
type Feed struct {
	// Thread ID
	id uint64
	// Message flushing ticker
	ticker
	// Buffer of unsent messages
	messageBuffer
	// Entire thread cached into memory
	cache threadCache
	// Add a client
	add chan common.Client
	// Remove client
	remove chan common.Client
	// Propagates mesages to all listeners
	send chan []byte
	// Insert a new post into the thread and propagate to listeners
	insertPost chan postCreationMessage
	// Insert an image into an already allocated post
	insertImage chan imageInsertionMessage
	// Send various simple messages targeted at a specific post
	sendPostMessage chan postMessage
	// Set body of an open post
	setOpenBody chan postBodyModMessage
	// Subscribed clients
	clients []common.Client
}

// Read existing posts into cache and start main loop
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
				f.clients = append(f.clients, c)
				if c.NewProtocol() {
					c.Send(f.cache.encodeThread(c.Last100()))
				} else {
					c.Send(f.cache.genSyncMessage())
				}
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
				case closePost:
					p := f.cache.Posts[msg.id]
					p.Editing = false
					f.cache.Posts[msg.id] = p
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
		Post: post.Post,
		msg:  msg,
	}
}

// Insert an image into an already allocated post
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
