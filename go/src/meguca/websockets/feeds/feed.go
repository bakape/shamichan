package feeds

import (
	"encoding/json"
	"meguca/common"
	"meguca/db"
)

// TODO: Propagate thread modetation events to all clients live

type postMessageType uint8

const (
	insertPost postMessageType = iota
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

type threadState struct {
	db.ThreadState
	DeletedImages []uint64 `json:"deletedImages"`
}

// A feed with synchronization logic of a certain thread
type Feed struct {
	// Thread ID
	id uint64
	// Message flushing ticker
	ticker
	// Buffer of unsent messages
	messageBuffer
	// Data used for synchronizing clients to the feed state.
	state threadState
	// Add a client
	add chan common.Client
	// Remove client
	remove chan common.Client
	// Propagates mesages to all listeners
	send chan []byte
	// Send various simple messages targeted at a specific post
	sendPostMessage chan postMessage
	// Subscribed clients
	clients []common.Client
}

// Read existing posts into cache and start main loop
func (f *Feed) Start() (err error) {
	f.state.ThreadState, err = db.GetThreadState(f.id)
	if err != nil {
		return
	}
	f.state.DeletedImages = make([]uint64, 0, 16)

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
				buf, _ := json.Marshal(f.state)
				c.Send(common.PrependMessageType(common.MessageConcat, buf))
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

			// Various post-related messages
			case msg := <-f.sendPostMessage:
				switch msg.typ {
				case insertPost:
					f.state.Replies = append(f.state.Replies, msg.id)
				case spoilerImage:
					f.state.Spoilered = append(f.state.Spoilered, msg.id)
				case ban:
					f.state.Banned = append(f.state.Banned, msg.id)
				case deletePost:
					f.state.Deleted = append(f.state.Deleted, msg.id)
				case deleteImage:
					f.state.DeletedImages = append(f.state.DeletedImages,
						msg.id)
				}
				f.bufferMessage(msg.msg)
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

// Small helper method
func (f *Feed) _sendPostMessage(typ postMessageType, id uint64, msg []byte) {
	f.sendPostMessage <- postMessage{
		typ: typ,
		id:  id,
		msg: msg,
	}
}

// Insert a new post into the thread or reclaim an open post after disconnect
// and propagate to listeners
func (f *Feed) InsertPost(id uint64, msg []byte) {
	f._sendPostMessage(insertPost, id, msg)
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
