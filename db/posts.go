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
	err = db.QueryRow("get_post_parenthood", id).Scan(&board, &op)
	return
}

// GetPostBoard retrieves the board of a post by ID
func GetPostBoard(id uint64) (board string, err error) {
	err = db.QueryRow("get_post_parenthood", id).Scan(&board)
	return
}

// Insert Post into thread and set its ID, creation time and moderation status.
// Thread OPs must have their post ID set to the thread ID.
// Any images are to be inserted in a separate call.
func InsertPost(tx *pgx.Tx, p *Post) (err error) {
	var res struct {
		ID         uint64
		Time       int64
		Moderation []common.ModerationEntry
	}
	err = tx.
		QueryRow(
			`insert_post`,
			p.OP,
			p.ID,
			p.Body,
			p.Flag,
			p.Name,
			p.Trip,
			p.Auth,
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
	p.Moderation = res.Moderation
	return
}

// GetPostPassword retrieves a post's modification password
func GetPostPassword(id uint64) (p []byte, err error) {
	err = db.QueryRow("get_post_password", id).Scan(&p)
	return
}

// SetPostCounter sets the post counter.
// Should only be used in tests.
func SetPostCounter(c uint64) error {
	_, err := db.Exec(`SELECT setval('post_id', $1)`, c)
	return err
}
