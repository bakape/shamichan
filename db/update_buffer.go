// Decrease the overhead of writing to the DB by bufferring post text body
// updates to every 0.2 seconds and dumping them in one transaction

package db

import (
	"bytes"
	"encoding/hex"
	"log"
	"sync"

	"github.com/bakape/meguca/util"
	"github.com/lib/pq"
)

var (
	toLog         = make(map[uint64]*MessageBuffer, 16)
	toReplaceBody = make(map[uint64]string, 32)
	bodyModCh     = make(chan bodyModRequest)
	bufPool       = sync.Pool{
		New: func() interface{} {
			return new(MessageBuffer)
		},
	}
)

type bodyModRequest struct {
	id, op uint64
	msg    []byte
	body   string
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

// Flush flushes the buffer into into a []byte and returns it together with the
// flushed message count. If no messages are stored, the returned buffer is nil.
// cpy specifies weather a copy of the underlying buffer should be returned
// instead. If not set, the caller must Reset() the buffer manually.
func (b *MessageBuffer) Flush(cpy bool) ([]byte, uint64) {
	if b.count == 0 {
		return nil, 0
	}

	flushed := b.count
	buf := b.buf.Bytes()
	b.count = 0

	if !cpy {
		return buf, flushed
	}
	c := make([]byte, len(buf))
	copy(c, buf)
	b.buf.Reset()
	return c, flushed
}

// Reset resets the internal buffer
func (b *MessageBuffer) Reset() {
	b.buf.Reset()
}

func init() {
	go func() {
		for {
			// Stop the timer, if there are no messages and resume on new ones.
			// Keeping the goroutine asleep reduces CPU usage.
			var flush util.PausableTicker
			flush.Start()

			select {
			case req := <-bodyModCh:
				flush.StartIfPaused()
				buf, ok := toLog[req.op]
				if !ok {
					buf = bufPool.Get().(*MessageBuffer)
					toLog[req.op] = buf
				}
				buf.Write(req.msg)
				toReplaceBody[req.id] = req.body
			case <-flush.C:
				if len(toLog) == 0 {
					flush.Pause()
					continue
				}
				if err := flushBodyUpdates(); err != nil {
					log.Printf("flushing body updates: %s\n", err)
				}
			}
		}
	}()
}

func flushBodyUpdates() error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer RollbackOnError(tx, &err)

	q := tx.Stmt(prepared["update_log_many"])
	for op, buf := range toLog {
		msg, _ := buf.Flush(false)
		split := bytes.Split(msg[2:], []byte{0})
		update := hex.EncodeToString(msg)
		_, err = q.Exec(op, pq.ByteaArray(split), update)

		buf.Reset()
		delete(toLog, op)
		bufPool.Put(buf)

		if err != nil {
			return err
		}
	}

	q = tx.Stmt(prepared["replace_body"])
	for id, body := range toReplaceBody {
		_, err = q.Exec(id, body)
		delete(toReplaceBody, id)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
