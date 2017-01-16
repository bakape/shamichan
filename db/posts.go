// Package db handles all core database interactions of the server
package db

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"

	"github.com/bakape/meguca/common"
	"github.com/lib/pq"
	"github.com/pquerna/ffjson/ffjson"
)

// DatabasePost is for writing new posts to a database. It contains the Password
// field, which is never exposed publically through Post.
type DatabasePost struct {
	Deleted bool
	common.StandalonePost
	Password []byte
	IP       string
}

// DatabaseThread is a template for writing new threads to the database
type DatabaseThread struct {
	ID                  uint64
	PostCtr, ImageCtr   uint32
	ReplyTime, BumpTime int64
	Subject, Board      string
	Log                 []string
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
	err = prepared["validateOP"].QueryRow(id, board).Scan(&valid)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return
}

// GetPostOP retrieves the parent thread ID of the passed post
func GetPostOP(id uint64) (op uint64, err error) {
	err = prepared["getPostOP"].QueryRow(id).Scan(&op)
	return
}

// BoardCounter retrieves the history or "progress" counter of a board
func BoardCounter(board string) (counter uint64, err error) {
	err = prepared["boardCounter"].QueryRow(board).Scan(&counter)
	return
}

// ThreadCounter retrieves the progress counter of a thread
func ThreadCounter(id uint64) (counter uint64, err error) {
	err = prepared["threadCounter"].QueryRow(id).Scan(&counter)
	return
}

// NewPostID reserves a new post ID
func NewPostID() (id uint64, err error) {
	err = prepared["newPostID"].QueryRow().Scan(&id)
	return id, err
}

// WritePost writes a post struct to database
func WritePost(tx *sql.Tx, p DatabasePost) error {
	ex := getExecutor(tx, "writePost")

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
			s, err := ffjson.Marshal(p.Commands[i])
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
func WriteThread(t DatabaseThread, p DatabasePost) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer rollbackOnError(tx, &err)

	_, err = tx.Stmt(prepared["writeOP"]).Exec(
		t.Board,
		pq.StringArray(t.Log),
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
	return tx.Commit()
}

// IsLocked returns if the thread is locked from posting
func IsLocked(id uint64) (bool, error) {
	var locked sql.NullBool
	err := prepared["isLocked"].QueryRow(id).Scan(&locked)
	return locked.Bool, err
}

// GetPostPassword retrieves a post's modification password
func GetPostPassword(id uint64) (p []byte, err error) {
	err = prepared["getPostPassword"].QueryRow(id).Scan(&p)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// HasImage returns, if the post has an image allocated
func HasImage(id uint64) (has bool, err error) {
	err = db.
		QueryRow(`
			SELECT true FROM posts
				WHERE id = $1 AND SHA1 IS NOT NULL`,
			id,
		).
		Scan(&has)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// GetIP returns an IP of the poster that created a post. Posts older than 7
// days will not have this field.
func GetIP(id uint64) (string, error) {
	var ip sql.NullString
	err := prepared["getIP"].QueryRow(id).Scan(&ip)
	return ip.String, err
}

// GetLog retrieves a slice of a thread's replication log
func GetLog(id, from, to uint64) ([][]byte, error) {
	var log pq.ByteaArray
	err := prepared["getLog"].QueryRow(id, from, to).Scan(&log)
	return [][]byte(log), err
}

// SetPostCounter sets the post counter. Should only be used for tests.
func SetPostCounter(c uint64) error {
	_, err := db.Exec(`SELECT setval('post_id', $1)`, c)
	return err
}
