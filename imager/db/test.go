package db

import (
	"context"

	"github.com/bakape/pg_util"
	"github.com/jackc/pgx/v4"
)

// DB functions only used for testing

// Insert sample thread and post into DB and return the post ID.
//
// Only used for testing.
func InsertSampleThread(pubKey uint64) (id uint64, err error) {
	err = InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
		q, args := pg_util.BuildInsert(pg_util.InsertOpts{
			Table: "threads",
			Data: struct {
				Subject string
				Tags    []string
			}{
				Subject: "test",
				Tags:    []string{"moe", "kyun"},
			},
			Suffix: "returning id",
		})
		err = tx.QueryRow(context.Background(), q, args...).Scan(&id)
		if err != nil {
			return
		}

		q, args = pg_util.BuildInsert(pg_util.InsertOpts{
			Table: "posts",
			Data: struct {
				ID        uint64
				PublicKey uint64 `db:"public_key"`
			}{
				ID:        id,
				PublicKey: pubKey,
			},
		})
		_, err = tx.Exec(context.Background(), q, args...)
		return
	})
	return
}
