package feeds

import (
	"encoding/json"
	"errors"
	"meguca/common"
	"meguca/db"
	"meguca/imager/assets"
	"meguca/util"
	"net/http"
)

// Terminates a running watcher. Only used in tests.
var terminateWatcher chan bool

// Serve thread updates over SSE for a set of threads
func WatchThreads(w http.ResponseWriter, r *http.Request, threads []uint64) (
	err error,
) {
	// Need to be able to flush for SSE
	fl, ok := w.(http.Flusher)
	if !ok {
		return errors.New("flushing not supported")
	}

	var closer <-chan bool
	if !db.IsTest { // httptest.ResponseRecord does not support closing
		// Returns a channel that blocks until the connection is closed
		cn, ok := w.(http.CloseNotifier)
		if !ok {
			return errors.New("closing not supported")
		}
		closer = cn.CloseNotify()
	}

	threads, err = db.FilterExistingThreads(threads...)
	if err != nil {
		return
	}

	h := w.Header()
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("Content-Type", "text/event-stream")

	wa := Watcher{
		w:       w,
		flusher: fl,

		// Prevents sending to Watcher from blocking the sender
		send:  make(chan []byte, 32),
		close: make(chan error, 2),
	}

	if !db.IsTest {
		// Forward external closing of connection
		go func() {
			<-closer
			wa.close <- nil
		}()
	}

	watchThreads(&wa, threads)
	defer unwatchThreads(&wa)

	err = wa.Start()
	if err != nil {
		err = util.WrapError("thread watcher", err)
	}
	return
}

// Client watching a thread for updates
type Watcher struct {
	w       http.ResponseWriter
	flusher http.Flusher
	send    chan []byte
	close   chan error
}

// Send message to watcher without blocking sender
func (w *Watcher) Send(msg []byte) {
	select {
	case w.send <- msg:
	default:
		select {
		case w.close <- errors.New("send buffer overflow"):
		default:
		}
	}
}

// Watcher event loop
func (w *Watcher) Start() (err error) {
	for {
		select {
		case msg := <-w.send:
			_, err = w.w.Write(msg)
			if err != nil {
				return
			}
			w.flusher.Flush()
		case err = <-w.close:
			return
		case <-terminateWatcher:
			return
		}
	}
}

type watcherMessage struct {
	ID    uint64        `json:"id"`
	OP    uint64        `json:"op"`
	Body  string        `json:"body"`
	Image string        `json:"image,omitempty"`
	Links []common.Link `json:"links,omitempty"`
}

// Encode mesage for server-sent events
func encodeSSEMessage(op uint64, p common.Post) ([]byte, error) {
	msg := watcherMessage{
		ID:    p.ID,
		OP:    op,
		Body:  p.Body,
		Links: p.Links,
	}
	if i := p.Image; i != nil && i.ThumbType != common.NoFile && !i.Spoiler {
		msg.Image = assets.RelativeThumbPath(i.ThumbType, i.SHA1)
	}
	buf, err := json.Marshal(msg)
	if err != nil {
		return nil, nil
	}
	b := make([]byte, 0, len(buf)+7)
	b = append(b, "data:"...)
	b = append(b, buf...)
	b = append(b, "\n\n"...)
	return b, nil
}
