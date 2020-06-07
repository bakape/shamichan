package db

import (
	"context"

	"github.com/bakape/pg_util"
	"github.com/jackc/pgx/v4"
)

// Common params for both post and thread insertion
type PostInsertParamsCommon struct {
	// Client authentication public key ID.
	//
	// Optional, to enable migrations of legacy data.
	PublicKey *uint64 `db:"public_key"`

	// Name set by poster
	Name *string

	// Tripcode
	Trip *string

	// Country flag to attach to poster
	Flag *string

	// Text body as JSON AST
	Body []byte
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

// Insert a new post into a specific thread. Returns post ID and page.
//
// params: either ReplyInsertParams or OPInsertparams.
func InsertPost(tx pgx.Tx, params interface{},
) (id uint64, page uint32, err error) {
	q, args := pg_util.BuildInsert(pg_util.InsertOpts{
		Table:  "posts",
		Data:   params,
		Suffix: "returning id, page",
	})
	err = tx.QueryRow(context.Background(), q, args...).Scan(&id, &page)
	return
}

// GetPost reads a single post from the database
func GetPost(ctx context.Context, id uint64) (post []byte, err error) {
	err = db.
		QueryRow(
			ctx,
			`select encode(p)
			from posts p
			where p.id = $1`,
			id,
		).
		Scan(&post)
	castNoRows(&post, &err)
	return
}

// Get thread and page numbers a post is in
func GetPostParenthood(id uint64) (thread uint64, page uint32, err error) {
	err = db.
		QueryRow(
			context.Background(),
			`select thread, page
			from posts
			where id = $1`,
			id,
		).
		Scan(&thread, &page)
	return
}
