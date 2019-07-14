// Package db handles all core database interactions of the server
package db

import (
	"database/sql"
	"encoding/json"

	"github.com/Masterminds/squirrel"
	"github.com/bakape/meguca/common"
)

// Post is for writing new posts to a database. It contains the Password
// field, which is never exposed publically through Post.
type Post struct {
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

// GetPostParenthood retrieves the board and OP of a post
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
	q := sq.Select("max(update_time) + count(*)").
		From("threads").
		Where("board = ?", board)
	return getCounter(q)
}

// AllBoardCounter retrieves the progress counter of the /all/ board
func AllBoardCounter() (uint64, error) {
	q := sq.Select("max(update_time) + count(*)").
		From("threads")
	return getCounter(q)
}

// WritePost writes a post struct to the database. Only used in tests and
// migrations.
func WritePost(tx *sql.Tx, p Post) (err error) {
	// Don't store empty strings of these in the database. Zero value != NULL.
	var (
		img, ip *string
		imgName string
		spoiler bool
	)
	if p.IP != "" {
		ip = &p.IP
	}
	if p.Image != nil {
		img = &p.Image.SHA1
		imgName = p.Image.Name
		spoiler = p.Image.Spoiler
	}

	_, err = sq.Insert("posts").
		Columns(
			"editing", "spoiler", "id", "board", "op", "time", "body", "flag",
			"name", "trip", "auth", "password", "ip",
			"SHA1", "imageName",
			"commands",
		).
		Values(
			p.Editing, spoiler, p.ID, p.Board, p.OP, p.Time, p.Body, p.Flag,
			p.Name, p.Trip, p.Auth, p.Password, ip,
			img, imgName,
			commandRow(p.Commands),
		).
		RunWith(tx).
		Exec()
	if err != nil {
		return
	}

	links := make([]uint64, len(p.Links))
	for id := range p.Links {
		links = append(links, id)
	}
	return writeLinks(tx, p.ID, links)
}

// Insert Post into thread and set its ID and creation time and moderation
// status.
// Thread OPs must have their post ID set to the thread ID.
// Any images are to be inserted in a separate call.
func InsertPost(tx *sql.Tx, p *Post) (err error) {
	args := make([]interface{}, 0, 12)
	args = append(args,
		p.Editing, p.Board, p.OP, p.Body, p.Flag,
		p.Name, p.Trip, p.Auth, p.Sage,
		p.Password, p.IP,
	)

	q := sq.Insert("posts").
		Columns(
			"editing", "board", "op", "body", "flag",
			"name", "trip", "auth", "sage",
			"password", "ip",
		)

	if p.ID != 0 { // OP of a thread
		q = q.Columns("id")
		args = append(args, p.ID)
	}

	var buf []byte
	err = q.
		Values(args...).
		Suffix("returning id, time, get_post_moderation(id)").
		RunWith(tx).
		QueryRow().
		Scan(&p.ID, &p.Time, &buf)
	if err != nil {
		return
	}
	if len(buf) != 0 {
		err = json.Unmarshal(buf, &p.Moderation)
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

// SetPostCounter sets the post counter.
// Should only be used in tests.
func SetPostCounter(c uint64) error {
	_, err := db.Exec(`SELECT setval('post_id', $1)`, c)
	return err
}
