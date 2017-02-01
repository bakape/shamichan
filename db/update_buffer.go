// Decrease the overhead of writing to the DB by bufferring post text body
// updates to every 0.2 seconds and dumping them in one transaction

package db

import (
	"log"

	"github.com/bakape/meguca/util"
	"github.com/lib/pq"
)

var (
	toLog         = make(map[uint64]*[][]byte, 16)
	toReplaceBody = make(map[uint64]string, 32)
	bodyModCh     = make(chan bodyModRequest)
)

type bodyModRequest struct {
	id, op uint64
	msg    []byte
	body   string
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
					b := make([][]byte, 0, 64)
					buf = &b
					toLog[req.op] = buf
				}
				*buf = append(*buf, req.msg)
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
		_, err = q.Exec(op, pq.ByteaArray(*buf))
		delete(toLog, op)
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
