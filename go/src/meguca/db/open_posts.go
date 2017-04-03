package db

import (
	"encoding/binary"

	"github.com/boltdb/bolt"
)

// SetOpenBody sets the open body of a post
func SetOpenBody(id uint64, body []byte) error {
	return boltDB.Batch(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("open_bodies")).Put(formatPostID(id), body)
	})
}

func formatPostID(id uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, id)
	return buf
}

// GetOpenBody retrieves an open body of a post
func GetOpenBody(id uint64) (body string, err error) {
	err = boltDB.View(func(tx *bolt.Tx) error {
		body = string(tx.Bucket([]byte("open_bodies")).Get(formatPostID(id)))
		return nil
	})
	return
}
