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
