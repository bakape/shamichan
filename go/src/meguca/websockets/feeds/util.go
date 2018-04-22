package feeds

import (
	"meguca/common"
	"time"
)

// TickerInterval sets the interval of ticker flushes
const TickerInterval = time.Millisecond * 100

// A time.Ticker that can be "paused"
type ticker struct {
	t *time.Ticker
	C <-chan time.Time
}

func (t *ticker) start() {
	t.t = time.NewTicker(TickerInterval)
	t.C = t.t.C
}

func (t *ticker) pause() {
	t.t.Stop()
	t.C = nil
}

func (t *ticker) startIfPaused() {
	if t.C == nil {
		t.start()
	}
}

// messageBuffer provides bufferring and concatenation for post update messages
type messageBuffer []string

// Write writes a message to b
func (b *messageBuffer) write(data []byte) {
	*b = append(*b, string(data))
}

// Flush flushes b into into a []byte and returns it.
// If no messages are stored, the returned buffer is nil.
func (b *messageBuffer) flush() []byte {
	if len(*b) == 0 {
		return nil
	}
	buf, _ := common.EncodeMessage(common.MessageConcat, *b)
	*b = (*b)[:0]
	return buf
}

// Embed for basic client event dispatching functionality
type baseFeed struct {
	// Add a client
	add chan common.Client
	// Remove client
	remove chan common.Client
	// Subscribed clients
	clients map[common.Client]bool
}

func (b *baseFeed) init() {
	b.add = make(chan common.Client)
	b.remove = make(chan common.Client)
	b.clients = make(map[common.Client]bool, 8)
}

func (b *baseFeed) addClient(c common.Client) {
	b.clients[c] = true
}

// If returned true, closing feed and parent listener loop should exit
func (b *baseFeed) removeClient(c common.Client) bool {
	delete(b.clients, c)
	if len(b.clients) != 0 {
		b.remove <- nil
		return false
	} else {
		b.remove <- c
		return true
	}
}

// Send a message to all connected clients
func (b *baseFeed) sendToAll(msg []byte) {
	for c := range b.clients {
		c.Send(msg)
	}
}
