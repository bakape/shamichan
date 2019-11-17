package db

import (
	"sort"
	"sync"

	"github.com/jackc/pgx"
)

var (
	// Buffer of pending open post body changes. Used to reduce DB I/O with
	// rapid open post body changes.
	openPostBodyBuffer = make(map[uint64]string)

	// Protects openPostBodyBuffer
	openPostBodyBufferMu sync.Mutex
)

// Flush any buffered open post bodies to DB
func FlushOpenPostBodies() (err error) {
	openPostBodyBufferMu.Lock()
	defer openPostBodyBufferMu.Unlock()

	if len(openPostBodyBuffer) == 0 {
		return
	}

	// Sort IDs for more sequential DB access
	toWrite := make(idSorter, 0, len(openPostBodyBuffer))
	for id := range openPostBodyBuffer {
		toWrite = append(toWrite, id)
	}
	sort.Sort(toWrite)

	return InTransaction(func(tx *pgx.Tx) (err error) {
		for _, id := range toWrite {
			_, err = tx.Exec(
				`update posts
				set body = $1
				where id = $2 and editing = true`,
				openPostBodyBuffer[id],
				id,
			)
			if err != nil {
				return
			}
			delete(openPostBodyBuffer, id)
		}
		return
	})
}

// Clear any buffered open post changes
func clearOpenPostBuffer() {
	openPostBodyBufferMu.Lock()
	defer openPostBodyBufferMu.Unlock()

	for k := range openPostBodyBuffer {
		delete(openPostBodyBuffer, k)
	}
}

// Buffer open post body for eventual writing to DB
func WriteOpenPostBody(id uint64, body string) {
	openPostBodyBufferMu.Lock()
	defer openPostBodyBufferMu.Unlock()

	openPostBodyBuffer[id] = body
}
