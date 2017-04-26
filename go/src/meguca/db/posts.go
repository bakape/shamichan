// Package db handles all core database interactions of the server
package db

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"meguca/common"
	"strconv"

	"github.com/lib/pq"
	"github.com/mailru/easyjson"
)

// Post is for writing new posts to a database. It contains the Password
// field, which is never exposed publically through Post.
type Post struct {
	Deleted bool
	common.StandalonePost
	Password []byte
	IP       string
}

// Thread is a template for writing new threads to the database
type Thread struct {
	ID                  uint64
	PostCtr, ImageCtr   uint32
	ReplyTime, BumpTime int64
	Subject, Board      string
}

// For decoding and encoding the tuple arrays we store links in
type linkRow [][2]uint64

func (l *linkRow) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		return l.scanBytes(src)
	case string:
		return l.scanBytes([]byte(src))
	case nil:
		*l = nil
		return nil
	default:
		return fmt.Errorf("db: cannot convert %T to [][2]uint", src)
	}
}

func (l *linkRow) scanBytes(src []byte) error {
	length := len(src)
	if length < 6 {
		return errors.New("db: source too short")
	}

	src = src[1 : length-1]

	// Determine needed size and preallocate final array
	commas := 0
	for _, b := range src {
		if b == ',' {
			commas++
		}
	}
	*l = make(linkRow, 0, (commas-1)/2+1)

	var (
		inner bool
		next  [2]uint64
		err   error
		buf   = make([]byte, 0, 16)
	)
	for _, b := range src {
		switch b {
		case '{': // New tuple
			inner = true
			buf = buf[0:0]
		case ',':
			if inner { // End of first uint
				next[0], err = strconv.ParseUint(string(buf), 10, 64)
				if err != nil {
					return err
				}
				buf = buf[0:0]
			}
		case '}': // End of tuple
			next[1], err = strconv.ParseUint(string(buf), 10, 64)
			if err != nil {
				return err
			}
			*l = append(*l, next)
		default:
			buf = append(buf, b)
		}
	}

	return nil
}

func (l linkRow) Value() (driver.Value, error) {
	n := len(l)
	if n == 0 {
		return nil, nil
	}

	b := make([]byte, 1, 16)
	b[0] = '{'
	for i, l := range l {
		if i != 0 {
			b = append(b, ',')
		}
		b = append(b, '{')
		b = strconv.AppendUint(b, l[0], 10)
		b = append(b, ',')
		b = strconv.AppendUint(b, l[1], 10)
		b = append(b, '}')
	}
	b = append(b, '}')

	return string(b), nil
}

// For encoding and decoding hash command results
type commandRow []common.Command

func (c *commandRow) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		return c.scanBytes(src)
	case string:
		return c.scanBytes([]byte(src))
	case nil:
		*c = nil
		return nil
	default:
		return fmt.Errorf("db: cannot convert %T to []common.Command", src)
	}
}

func (c *commandRow) scanBytes(data []byte) (err error) {
	var bArr pq.ByteaArray
	err = bArr.Scan(data)
	if err != nil {
		return
	}

	*c = make([]common.Command, len(bArr))
	for i := range bArr {
		err = (*c)[i].UnmarshalJSON(bArr[i])
		if err != nil {
			return
		}
	}

	return
}

func (c commandRow) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}

	var strArr = make(pq.StringArray, len(c))
	for i := range strArr {
		s, err := easyjson.Marshal(c[i])
		if err != nil {
			return nil, err
		}
		strArr[i] = string(s)
	}

	return strArr.Value()
}

// ValidateOP confirms the specified thread exists on specific board
func ValidateOP(id uint64, board string) (valid bool, err error) {
	err = prepared["validate_op"].QueryRow(id, board).Scan(&valid)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return
}

// GetPostOP retrieves the parent thread ID of the passed post
func GetPostOP(id uint64) (op uint64, err error) {
	err = prepared["get_post_op"].QueryRow(id).Scan(&op)
	return
}

// GetThreadBoard retrieves the board of a thread by id
func GetThreadBoard(id uint64) (string, error) {
	return getPostBoard(id, "get_thread_board")
}

func getPostBoard(id uint64, queryID string) (board string, err error) {
	err = prepared[queryID].QueryRow(id).Scan(&board)
	return
}

// GetPostBoard retrieves the board of a post by ID
func GetPostBoard(id uint64) (string, error) {
	return getPostBoard(id, "get_post_board")
}

// PostCounter retrieves the current post counter
func PostCounter() (uint64, error) {
	return getCounter("post_counter")
}

func getCounter(queryID string, args ...interface{}) (uint64, error) {
	var c sql.NullInt64
	err := prepared[queryID].QueryRow(args...).Scan(&c)
	return uint64(c.Int64), err
}

// BoardCounter retrieves the progress counter of a board
func BoardCounter(board string) (uint64, error) {
	return getCounter("board_counter", board)
}

// AllBoardCounter retrieves the progress counter of the /all/ board
func AllBoardCounter() (uint64, error) {
	return getCounter("all_board_counter")
}

// ThreadCounter retrieves the progress counter of a thread
func ThreadCounter(id uint64) (uint64, error) {
	return getCounter("thread_counter", id)
}

// NewPostID reserves a new post ID
func NewPostID() (id uint64, err error) {
	err = prepared["new_post_id"].QueryRow().Scan(&id)
	return id, err
}

// InsertPost inserts a post into an existing thread
func InsertPost(p Post, sage bool) (err error) {
	err = execPrepared("insert_post", append(genPostCreationArgs(p), sage)...)
	if err != nil {
		return
	}

	if p.Editing {
		err = SetOpenBody(p.ID, []byte(p.Body))
	}

	return
}

func genPostCreationArgs(p Post) []interface{} {
	// Don't store empty strings in the database. Zero value != NULL.
	var (
		name, trip, auth, img, imgName, ip *string
		spoiler                            bool
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
	if p.Image != nil {
		img = &p.Image.SHA1
		imgName = &p.Image.Name
		spoiler = p.Image.Spoiler
	}

	return []interface{}{
		p.Editing, spoiler, p.ID, p.Board, p.OP, p.Time, p.Body, name, trip,
		auth, p.Password, ip, img, imgName,
		linkRow(p.Links), linkRow(p.Backlinks), commandRow(p.Commands),
	}
}

// WritePost writes a post struct to the database. Only used in tests and
// migrations.
func WritePost(tx *sql.Tx, p Post) (err error) {
	_, err = getExecutor(tx, "write_post").Exec(genPostCreationArgs(p)...)
	if err != nil {
		return
	}

	if p.Editing {
		err = SetOpenBody(p.ID, []byte(p.Body))
	}

	return
}

// InsertThread inserts a new thread into the database
func InsertThread(subject string, p Post) (err error) {
	imgCtr := 0
	if p.Image != nil {
		imgCtr = 1
	}
	err = execPrepared(
		"insert_thread",
		append([]interface{}{subject, imgCtr}, genPostCreationArgs(p)...)...,
	)
	if err != nil {
		return
	}

	if p.Editing {
		err = SetOpenBody(p.ID, []byte(p.Body))
	}

	return
}

// WriteThread writes a thread and it's OP to the database. Only used for tests
// and migrations.
func WriteThread(tx *sql.Tx, t Thread, p Post) (err error) {
	passedTx := tx != nil
	if !passedTx {
		tx, err = db.Begin()
		if err != nil {
			return err
		}
		defer RollbackOnError(tx, &err)
	}

	_, err = tx.Stmt(prepared["write_op"]).Exec(
		t.Board,
		t.ID,
		t.PostCtr,
		t.ImageCtr,
		t.ReplyTime,
		t.BumpTime,
		t.Subject,
	)
	if err != nil {
		return err
	}

	err = WritePost(tx, p)
	if err != nil {
		return err
	}

	if !passedTx {
		return tx.Commit()
	}
	return nil
}

// GetPostPassword retrieves a post's modification password
func GetPostPassword(id uint64) (p []byte, err error) {
	err = prepared["get_post_password"].QueryRow(id).Scan(&p)
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
