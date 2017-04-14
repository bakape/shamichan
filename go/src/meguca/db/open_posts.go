package db

import (
	"database/sql"
	"encoding/binary"

	"github.com/boltdb/bolt"
)

// SetOpenBody sets the open body of a post
func SetOpenBody(id uint64, body []byte) error {
	return boltDB.Batch(func(tx *bolt.Tx) error {
		return bodyBucket(tx).Put(formatPostID(id), body)
	})
}

func bodyBucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket([]byte("open_bodies"))
}

func formatPostID(id uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, id)
	return buf
}

// GetOpenBody retrieves an open body of a post
func GetOpenBody(id uint64) (body string, err error) {
	err = boltDB.View(func(tx *bolt.Tx) error {
		body = string(bodyBucket(tx).Get(formatPostID(id)))
		return nil
	})
	return
}

func deleteOpenPostBody(id uint64) error {
	return boltDB.Batch(func(tx *bolt.Tx) error {
		return bodyBucket(tx).Delete(formatPostID(id))
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
	return boltDB.Update(func(tx *bolt.Tx) (err error) {
		buc := bodyBucket(tx)
		for _, id := range toDelete {
			err = buc.Delete(formatPostID(id))
			if err != nil {
				return
			}
		}
		return
	})
}
