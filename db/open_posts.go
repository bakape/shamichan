package db

import (
	"database/sql"
	"sort"
	"sync"
)

var (
	// Buffer of pending open post body changes. Used to reduce DB I/O with
	// rapid open post body changes.
	openPostBodyBuffer = make(map[uint64]string)

	// Protects openPostBodyBuffer
	openPostBodyBufferBuffer sync.Mutex
)

// Flush any buffered open post bodies to DB
func FlushOpenPostBodies() (err error) {
	openPostBodyBufferBuffer.Lock()
	defer openPostBodyBufferBuffer.Unlock()

	if len(openPostBodyBuffer) == 0 {
		return
	}

	// Sort IDs for more sequential DB access
	toWrite := make(idSorter, 0, len(openPostBodyBuffer))
	for id := range openPostBodyBuffer {
		toWrite = append(toWrite, id)
	}
	sort.Sort(toWrite)

	return InTransaction(func(tx *sql.Tx) (err error) {
		q, err := tx.Prepare(
			`update posts
			set body = $1
			where id = $2 and editing = true`,
		)
		if err != nil {
			return
		}

		for _, id := range toWrite {
			_, err = q.Exec(openPostBodyBuffer[id], id)
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
	openPostBodyBufferBuffer.Lock()
	defer openPostBodyBufferBuffer.Unlock()

	for k := range openPostBodyBuffer {
		delete(openPostBodyBuffer, k)
	}
}

// Buffer open post body for eventual writing to DB
func WriteOpenPostBody(id uint64, body string) {
	openPostBodyBufferBuffer.Lock()
	defer openPostBodyBufferBuffer.Unlock()

	openPostBodyBuffer[id] = body
}
