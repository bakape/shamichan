package db

import (
	"database/sql"
	"encoding/binary"

	"github.com/boltdb/bolt"
)

// SetOpenBody sets the open body of a post
func SetOpenBody(id uint64, body []byte) error {
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
// buffer must persist till the end of the transaction, this is needed.
func encodeUint64Heap(i uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, i)
	return buf
}

// GetOpenBody retrieves an open body of a post
func GetOpenBody(id uint64) (body string, err error) {
	buf := encodeUint64(id)
	err = boltDB.View(func(tx *bolt.Tx) error {
		body = string(bodyBucket(tx).Get(buf[:]))
		return nil
	})
	return
}

func deleteOpenPostBody(id uint64) error {
	buf := encodeUint64(id)
	return boltDB.Batch(func(tx *bolt.Tx) error {
		return bodyBucket(tx).Delete(buf[:])
	})
}

// Delete orphaned post bodies, that refer to posts already closed or deleted.
// This can happen on server restarts, board deletion, etc.
func cleanUpOpenPostBodies() (err error) {
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
	tx, err := db.Begin()
	if err != nil {
		return
	}
	err = setReadOnly(tx)
	if err != nil {
		return
	}
	defer tx.Rollback()

	var (
		isOpen bool
		q      = tx.Stmt(prepared["is_open_post"])
	)
	for _, id := range ids {
		err = q.QueryRow(id).Scan(&isOpen)
		if err != nil {
			if err == sql.ErrNoRows {
				err = nil
			} else {
				return
			}
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
}
