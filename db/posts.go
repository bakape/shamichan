// Package db handles all core database interactions of the server
package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/bakape/meguca/common"
	"github.com/lib/pq"
)

var (
	// ErrTooManyMessages denotes too many messages have been requested from
	// the replication log
	ErrTooManyMessages = errors.New("too many messages requested")
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
	Log                 [][]byte
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
		return fmt.Errorf("db: source too short")
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

// PostCounter retrieves the current post counter
func PostCounter() (uint64, error) {
	var c sql.NullInt64
	err := prepared["post_counter"].QueryRow().Scan(&c)
	return uint64(c.Int64), err
}

// BoardCounter retrieves the history or "progress" counter of a board
func BoardCounter(board string) (uint64, error) {
	var c sql.NullInt64
	err := prepared["board_counter"].QueryRow(board).Scan(&c)
	return uint64(c.Int64), err
}

// ThreadCounter retrieves the progress counter of a thread
func ThreadCounter(id uint64) (uint64, error) {
	var c sql.NullInt64
	err := prepared["thread_counter"].QueryRow(id).Scan(&c)
	return uint64(c.Int64), err
}

// NewPostID reserves a new post ID
func NewPostID() (id uint64, err error) {
	err = prepared["new_post_id"].QueryRow().Scan(&id)
	return id, err
}

// WritePost writes a post struct to database
func WritePost(tx *sql.Tx, p Post) error {
	ex := getExecutor(tx, "write_post")

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

	var comm pq.StringArray
	if p.Commands != nil {
		comm = make(pq.StringArray, len(p.Commands))
		for i := range comm {
			s, err := json.Marshal(p.Commands[i])
			if err != nil {
				return err
			}
			comm[i] = string(s)
		}
	}

	_, err := ex.Exec(
		p.Editing, spoiler, p.ID, p.Board, p.OP, p.Time, p.Body, name, trip,
		auth, p.Password, ip, img, imgName, linkRow(p.Links),
		linkRow(p.Backlinks), comm,
	)
	return err
}

// WriteThread writes a thread and it's OP to the database
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
		pq.ByteaArray(t.Log),
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

// IsLocked returns if the thread is locked from posting
func IsLocked(id uint64) (bool, error) {
	var locked sql.NullBool
	err := prepared["is_locked"].QueryRow(id).Scan(&locked)
	return locked.Bool, err
}

// GetPostPassword retrieves a post's modification password
func GetPostPassword(id uint64) (p []byte, err error) {
	err = prepared["get_post_password"].QueryRow(id).Scan(&p)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// GetIP returns an IP of the poster that created a post. Posts older than 7
// days will not have this field.
func GetIP(id uint64) (string, error) {
	var ip sql.NullString
	err := prepared["get_ip"].QueryRow(id).Scan(&ip)
	return ip.String, err
}

// GetLog retrieves a slice of a thread's replication log
func GetLog(id, from, to uint64) ([][]byte, error) {
	if to-from > 500 {
		return nil, ErrTooManyMessages
	}
	var log pq.ByteaArray
	err := prepared["get_log"].QueryRow(id, from, to).Scan(&log)
	return [][]byte(log), err
}

// SetPostCounter sets the post counter. Should only be used in tests.
func SetPostCounter(c uint64) error {
	_, err := db.Exec(`SELECT setval('post_id', $1)`, c)
	return err
}

// DeletePosts marks the target posts as deleted
func DeletePosts(board string, ids ...uint64) error {
	return execPrepared("delete_posts", encodeIDArray(ids...), board)
}

func encodeIDArray(ids ...uint64) string {
	b := make([]byte, 1, 16)
	b[0] = '{'
	for i, id := range ids {
		if i != 0 {
			b = append(b, ',')
		}
		b = strconv.AppendUint(b, id, 10)
	}
	b = append(b, '}')
	return string(b)
}

// SpoilerImage spoilers an already allocated image
func SpoilerImage(id uint64) (err error) {
	op, err := GetPostOP(id)
	if err != nil {
		return
	}

	msg, err := common.EncodeMessage(common.MessageSpoiler, id)
	if err != nil {
		return
	}
	return updatePost(id, op, msg, "spoiler_image", nil)
}

// BumpThread dumps up thread counters and adds a message to the thread's
// replication log. tx must not be nil.
func BumpThread(
	tx *sql.Tx,
	id uint64,
	reply, bump, image bool,
	msg []byte,
) error {
	_, err := tx.Stmt(prepared["bump_thread"]).Exec(id, reply, bump, image)
	if err != nil {
		return err
	}
	return UpdateLog(tx, id, msg)
}

// BumpBoard increment's a board's progress counter. tx must not be nil.
func BumpBoard(tx *sql.Tx, board string) error {
	_, err := tx.Stmt(prepared["bump_board"]).Exec(board)
	return err
}
