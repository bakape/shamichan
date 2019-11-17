// Package db handles all core database interactions of the server
package db

import (
	"github.com/bakape/meguca/common"
	"github.com/jackc/pgx"
)

// Post is for writing new posts to a database. It contains the Password
// field, which is never exposed publically through Post.
type Post struct {
	common.StandalonePost
	Password []byte
	IP       string
}

// GetPostParenthood retrieves the board and OP of a post
func GetPostParenthood(id uint64) (board string, op uint64, err error) {
	err = db.
		QueryRow(
			`select board, op
			from posts p
			join threads t on t.id = p.op
			where p.id = $1`,
			id,
		).
		Scan(&board, &op)
	return
}

// GetPostBoard retrieves the board of a post by ID
func GetPostBoard(id uint64) (board string, err error) {
	err = db.
		QueryRow(
			`select board, op
			from posts p
			join threads t on t.id = p.op
			where p.id = $1`,
			id,
		).
		Scan(&board)
	return
}

// Insert Post into thread and set its ID, creation time and moderation status.
// Thread OPs must have their post ID set to the thread ID.
// Any images are to be inserted in a separate call.
func InsertPost(tx *pgx.Tx, p *Post) (err error) {
	var res struct {
		ID, Page uint64
		Time     int64
	}
	err = tx.
		QueryRow(
			`select insert_image(
				$1::bigint,
				$2::char(86),
				$3::varchar(200),
				$4::bool
			)`,
			p.OP,
			p.ID,
			p.Body,
			p.Flag,
			p.Name,
			p.Trip,
			p.Sage,
			p.Password,
			p.IP,
		).
		Scan(&res)
	if err != nil {
		return
	}
	p.ID = res.ID
	p.Time = res.Time
	p.Page = res.Page
	return
}
