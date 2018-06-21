// Package db handles all core database interactions of the server
package db

import (
	"database/sql"
	"meguca/common"

	"github.com/Masterminds/squirrel"
)

// Post is for writing new posts to a database. It contains the Password
// field, which is never exposed publically through Post.
type Post struct {
	Deleted bool
	common.StandalonePost
	Password []byte
	IP       string
}

func selectPost(id uint64, columns ...string) rowScanner {
	return sq.Select(columns...).
		From("posts").
		Where("id = ?", id).
		QueryRow()
}

// GetPostOP retrieves the parent thread ID of the passed post
func GetPostOP(id uint64) (op uint64, err error) {
	err = selectPost(id, "op").Scan(&op)
	return
}

// Retrieve the board and OP of a post
func GetPostParenthood(id uint64) (board string, op uint64, err error) {
	err = selectPost(id, "board", "op").Scan(&board, &op)
	return
}

// GetPostBoard retrieves the board of a post by ID
func GetPostBoard(id uint64) (board string, err error) {
	err = selectPost(id, "board").Scan(&board)
	return
}

func getCounter(q squirrel.SelectBuilder) (uint64, error) {
	var c sql.NullInt64
	err := q.QueryRow().Scan(&c)
	return uint64(c.Int64), err
}

// BoardCounter retrieves the progress counter of a board
func BoardCounter(board string) (uint64, error) {
	q := sq.Select("max(replyTime) + count(*)").
		From("threads").
		Where("board = ?", board)
	return getCounter(q)
}

// AllBoardCounter retrieves the progress counter of the /all/ board
func AllBoardCounter() (uint64, error) {
	q := sq.Select("max(replyTime) + count(*)").
		From("threads")
	return getCounter(q)
}

// NewPostID reserves a new post ID
func NewPostID(tx *sql.Tx) (id uint64, err error) {
	err = tx.QueryRow(`select nextval('post_id')`).Scan(&id)
	return id, err
}

// WritePost writes a post struct to the database. Only used in tests and
// migrations.
// bumpReplyTime: increment thread replyTime
// sage: don't increment bumpTime
func WritePost(tx *sql.Tx, p Post, bumpReplyTime, sage bool) (err error) {
	// Don't store empty strings in the database. Zero value != NULL.
	var (
		name, trip, auth, img, imgName, ip, flag, posterID *string
		spoiler                                            bool
	)
	if p.Name != "" {
		name = &p.Name
	}
	if p.Trip != "" {
		trip = &p.Trip
	}
	if p.Auth != "" {
		auth = &p.Auth
	}
	if p.IP != "" {
		ip = &p.IP
	}
	if p.Flag != "" {
		flag = &p.Flag
	}
	if p.PosterID != "" {
		posterID = &p.PosterID
	}
	if p.Image != nil {
		img = &p.Image.SHA1
		imgName = &p.Image.Name
		spoiler = p.Image.Spoiler
	}

	q := sq.Insert("posts").
		Columns(
			"editing", "spoiler", "id", "board", "op", "time", "body", "flag",
			"posterID", "name", "trip", "auth", "password", "ip",
			"SHA1", "imageName",
			"commands",
		).
		Values(
			p.Editing, spoiler, p.ID, p.Board, p.OP, p.Time, p.Body, flag,
			posterID, name, trip, auth, p.Password, ip, img, imgName,
			commandRow(p.Commands),
		)
	err = withTransaction(tx, q).Exec()
	if err != nil {
		return
	}
	err = writeLinks(tx, p.ID, p.Links)
	if err != nil {
		return
	}
	if bumpReplyTime {
		err = bumpThread(tx, p.OP, !sage)
		if err != nil {
			return
		}
	}

	if p.Editing {
		err = SetOpenBody(p.ID, []byte(p.Body))
	}
	return
}

// GetPostPassword retrieves a post's modification password
func GetPostPassword(id uint64) (p []byte, err error) {
	err = sq.Select("password").From("posts").Where("id = ?", id).Scan(&p)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// SetPostCounter sets the post counter. Should only be used in tests.
func SetPostCounter(c uint64) error {
	_, err := db.Exec(`SELECT setval('post_id', $1)`, c)
	return err
}
