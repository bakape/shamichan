package feeds

import (
	"meguca/util"
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
type messageBuffer []byte

// Write writes a message to b
func (b *messageBuffer) write(data []byte) {
	if len(*b) == 0 {
		*b = append(*b, "33"...)
	} else {
		*b = append(*b, 0)
	}
	*b = append(*b, data...)
}

// Flush flushes b into into a []byte and returns it.
// If no messages are stored, the returned buffer is nil.
func (b *messageBuffer) flush() []byte {
	if len(*b) == 0 {
		return nil
	}

	// Need to copy, because buffer will be sent to multiple threads
	buf := util.CloneBytes(*b)
	*b = (*b)[:0]
	return buf
}
