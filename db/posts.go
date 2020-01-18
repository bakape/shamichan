package db

import (
	"context"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/pg_util"
	"github.com/jackc/pgx/v4"
)

// Common params for both post and thread insertion
type PostInsertParamsCommon struct {
	// Client authorization key
	AuthKey auth.AuthKey `db:"auth_key"`
}

// For inserting a thread reply
type ReplyInsertParams struct {
	// Post was saged
	Sage bool

	// Parent thread
	Thread uint64

	PostInsertParamsCommon
}

// For inserting the OP of a thread
type OPInsertparams struct {
	// New post ID
	ID uint64

	ReplyInsertParams
}

// Insert a new post into a specific thread. Returns post ID.
//
// params: either ReplyInsertParams or OPInsertparams.
func InsertPost(
	ctx context.Context,
	tx pgx.Tx,
	params interface{},
) (id uint64, err error) {
	q, args := pg_util.BuildInsert(pg_util.InsertOpts{
		Table:  "posts",
		Data:   params,
		Suffix: "returning id",
	})
	err = tx.QueryRow(ctx, q, args...).Scan(&id)
	return
}
