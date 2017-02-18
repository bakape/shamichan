// Package util contains various general utility functions used throughout
// the project.
package util

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"time"
)

// PausableTicker is a time.Ticker that can be paused
type PausableTicker struct {
	t *time.Ticker
	C <-chan time.Time
}

// Start starts p
func (p *PausableTicker) Start() {
	p.t = time.NewTicker(time.Millisecond * 50)
	p.C = p.t.C
}

// Pause pauses p
func (p *PausableTicker) Pause() {
	p.t.Stop()
	p.C = nil
}

// StartIfPaused start p back up, if p is paused
func (p *PausableTicker) StartIfPaused() {
	if p.C == nil {
		p.Start()
	}
}

// MessageBuffer provides bufferring and concatenation for post update messages
type MessageBuffer struct {
	buf   bytes.Buffer
	count uint64
}

// Write writes a message to b
func (b *MessageBuffer) Write(data []byte) {
	if b.count == 0 {
		b.buf.WriteString("33")
	} else {
		b.buf.WriteByte(0)
	}
	b.count++
	b.buf.Write(data)
}

// Flush flushes b into into a []byte and returns it together with the flushed
// message count. If no messages are stored, the returned buffer is nil.
func (b *MessageBuffer) Flush() ([]byte, uint64) {
	if b.count == 0 {
		return nil, 0
	}

	flushed := b.count
	buf := b.buf.Bytes()
	b.count = 0

	c := make([]byte, len(buf))
	copy(c, buf)
	b.buf.Reset()
	return c, flushed
}

// WrapError wraps error types to create compound error chains
func WrapError(text string, err error) error {
	return wrappedError{
		text:  text,
		inner: err,
	}
}

type wrappedError struct {
	text  string
	inner error
}

func (e wrappedError) Error() string {
	text := e.text
	if e.inner != nil {
		text += ": " + e.inner.Error()
	}
	return text
}

// Waterfall executes a slice of functions until the first error returned. This
// error, if any, is returned to the caller.
func Waterfall(fns ...func() error) (err error) {
	for _, fn := range fns {
		err = fn()
		if err != nil {
			break
		}
	}
	return
}

// HashBuffer computes a base64 MD5 hash from a buffer
func HashBuffer(buf []byte) string {
	hash := md5.Sum(buf)
	return base64.RawStdEncoding.EncodeToString(hash[:])
}

// ConcatStrings efficiently concatenates strings with only one extra allocation
func ConcatStrings(s ...string) string {
	l := 0
	for _, s := range s {
		l += len(s)
	}
	b := make([]byte, 0, l)
	for _, s := range s {
		b = append(b, s...)
	}
	return string(b)
}
