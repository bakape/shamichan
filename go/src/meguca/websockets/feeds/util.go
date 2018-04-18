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
	clients []common.Client
}

func (b *baseFeed) init() {
	b.add = make(chan common.Client)
	b.remove = make(chan common.Client)
	b.clients = make([]common.Client, 0, 8)
}

func (b *baseFeed) addClient(c common.Client) {
	b.clients = append(b.clients, c)
}

// If returned true, closing feed and parent listener loop should exit
func (b *baseFeed) removeClient(c common.Client) bool {
	for i, cl := range b.clients {
		if cl == c {
			copy(b.clients[i:], b.clients[i+1:])
			b.clients[len(b.clients)-1] = nil
			b.clients = b.clients[:len(b.clients)-1]
			break
		}
	}
	if len(b.clients) != 0 {
		b.remove <- nil
		return false
	} else {
		b.remove <- c
		return true
	}
}
