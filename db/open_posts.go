package db

import (
	"database/sql"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Chiiruno/meguca/common"
	"github.com/boltdb/bolt"
)

const (
	boltNotOpened = iota //  Not opened yet in this server instance
	boltDBOpen           // Opened and ready fort operation
	boltDBClosed         // Closed for graceful restart
)

var (
	// Current state of BoltDB database.
	// Should only be accessed using atomic operations.
	boltDBState uint32

	// Ensures boltdb is opened only once
	boltDBOnce sync.Once
)

// Close DB and release resources
func Close() (err error) {
	atomic.StoreUint32(&boltDBState, boltDBClosed)
	return boltDB.Close()
}

// Need to drop any incoming requests, when Db is closed during graceful restart
func boltDBisOpen() bool {
	return atomic.LoadUint32(&boltDBState) == boltDBOpen
}

// Open boltdb, only when needed. This helps preventing conflicts on swapping
// the database accessing process during graceful restarts.
// If boltdb has already been closed, return open=false.
func tryOpenBoltDB() (open bool, err error) {
	boltDBOnce.Do(func() {
		boltDB, err = bolt.Open(
			"db.db",
			0600,
			&bolt.Options{
				Timeout: time.Second,
			})
		if err != nil {
			return
		}

		err = boltDB.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte("open_bodies"))
			return err
		})
		if err != nil {
			return
		}

		atomic.StoreUint32(&boltDBState, boltDBOpen)
		return
	})
	if err != nil {
		return
	}

	open = boltDBisOpen()
	return
}

// SetOpenBody sets the open body of a post
func SetOpenBody(id uint64, body []byte) (err error) {
	ok, err := tryOpenBoltDB()
	if err != nil || !ok {
		return
	}

	buf := encodeUint64(id)
	return boltDB.Batch(func(tx *bolt.Tx) error {
		return bodyBucket(tx).Put(buf[:], body)
	})
}

func bodyBucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket([]byte("open_bodies"))
}

// Encode uint64 for storage in BoltDB without heap allocations
func encodeUint64(i uint64) [8]byte {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], i)
	return buf
}

// Same as encodeUint64, but allocates on the heap. In some cases, where the
// buffer must persist after the end of the transaction, this is needed.
func encodeUint64Heap(i uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, i)
	return buf
}

// GetOpenBody retrieves an open body of a post
func GetOpenBody(id uint64) (body string, err error) {
	ok, err := tryOpenBoltDB()
	if err != nil || !ok {
		return
	}

	buf := encodeUint64(id)
	err = boltDB.View(func(tx *bolt.Tx) error {
		body = string(bodyBucket(tx).Get(buf[:]))
		return nil
	})
	return
}

func deleteOpenPostBody(id uint64) (err error) {
	ok, err := tryOpenBoltDB()
	if err != nil || !ok {
		return
	}

	buf := encodeUint64(id)
	return boltDB.Batch(func(tx *bolt.Tx) error {
		return bodyBucket(tx).Delete(buf[:])
	})
}

// Inject open post bodies from the embedded database into the posts
func injectOpenBodies(posts []*common.Post) (err error) {
	if len(posts) == 0 {
		return
	}

	ok, err := tryOpenBoltDB()
	if err != nil || !ok {
		return
	}

	tx, err := boltDB.Begin(false)
	if err != nil {
		return
	}

	buc := tx.Bucket([]byte("open_bodies"))
	for _, p := range posts {
		p.Body = string(buc.Get(encodeUint64Heap(p.ID)))
	}

	return tx.Rollback()
}

// Delete orphaned post bodies, that refer to posts already closed or deleted.
// This can happen on server restarts, board deletion, etc.
func cleanUpOpenPostBodies() (err error) {
	ok, err := tryOpenBoltDB()
	if err != nil || !ok {
		return
	}

	// Read IDs of all post bodies
	var ids []uint64
	err = boltDB.View(func(tx *bolt.Tx) error {
		buc := bodyBucket(tx)
		ids = make([]uint64, 0, buc.Stats().KeyN)
		return buc.ForEach(func(k, _ []byte) error {
			ids = append(ids, binary.LittleEndian.Uint64(k))
			return nil
		})
	})
	if err != nil {
		return
	}

	// Find bodies with closed parents
	toDelete := make([]uint64, 0, len(ids))
	return InTransaction(true, func(tx *sql.Tx) (err error) {
		var isOpen bool
		q, err := tx.Prepare(`select 'true' from posts
			where id = $1 and editing = 'true'`)
		if err != nil {
			return
		}
		for _, id := range ids {
			err = q.QueryRow(id).Scan(&isOpen)
			switch err {
			case nil:
			case sql.ErrNoRows:
				err = nil
				isOpen = false // Treat missing as closed
			default:
				return
			}
			if !isOpen {
				toDelete = append(toDelete, id)
			}
		}

		// Delete closed post bodies, if any
		if len(toDelete) == 0 {
			return
		}
		return boltDB.Batch(func(tx *bolt.Tx) (err error) {
			buc := bodyBucket(tx)
			for _, id := range toDelete {
				err = buc.Delete(encodeUint64Heap(id))
				if err != nil {
					return
				}
			}
			return
		})
	})
}
