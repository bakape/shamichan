package db

import (
	"context"

	"github.com/bakape/pg_util"
	"github.com/go-playground/log"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// InTransaction runs a function inside a transaction and handles comminting and
// rollback on error.
func InTransaction(
	ctx context.Context,
	fn func(tx pgx.Tx) (err error),
) error {
	return pg_util.InTransaction(ctx, db, fn)
}

// IsConflictError returns if an error is a unique key conflict error
func IsConflictError(err error) bool {
	return extractException(err) == "unique_violation"
}

// Listen assigns a function to listen to Postgres notifications on a channel.
func Listen(opts pg_util.ListenOpts) (err error) {
	opts.ConnectionURL = connectionURL
	opts.OnError = func(err error) {
		log.Error(err)
	}
	return pg_util.Listen(opts)
}

// Try to extract an exception message, if err is *pq.Error
func extractException(err error) string {
	if err, ok := err.(*pgconn.PgError); ok {
		return err.Message
	}
	return ""
}
