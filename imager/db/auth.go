package db

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/jackc/pgconn"
	uuid "github.com/satori/go.uuid"
)

// Write public key to DB, if not already written.
// Return its private and public IDs and, if this was a fresh insert or an
// existing key.
func RegisterPublicKey(pubKey []byte) (
	privID uint64,
	pubID uuid.UUID,
	fresh bool,
	err error,
) {
	var (
		n   int
		tag pgconn.CommandTag
	)

try:
	n, err = rand.Reader.Read(pubID[:])
	if err != nil {
		return
	}
	if n != 16 {
		err = fmt.Errorf("incomplete random buffer read: %d", n)
		return
	}

	// Perform upsert attempt first to ensure public key is always in the DB by
	// the time the select is executed
	tag, err = db.Exec(
		context.Background(),
		`insert into public_keys (public_id, public_key)
		values ($1, $2)
		on conflict (public_key) do nothing`,
		pubID, pubKey,
	)
	if err != nil {
		if err, ok := err.(*pgconn.PgError); ok &&
			err.Message == "unique_violation" &&
			err.ColumnName == "public_id" {
			goto try
		}
		return
	}
	fresh = tag.RowsAffected() == 1

	err = db.
		QueryRow(
			context.Background(),
			`select id, public_id
			from public_keys
			where public_key = $1`,
			pubKey,
		).
		Scan(&privID, &pubID)
	return
}

// Get public key by its public ID
func GetPubKey(pubID uuid.UUID) (privID uint64, pubKey []byte, err error) {
	err = db.
		QueryRow(
			context.Background(),
			`select id, public_key
			from public_keys
			where public_id = $1`,
			pubID,
		).
		Scan(&privID, &pubKey)
	return
}
