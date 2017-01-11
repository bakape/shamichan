// Package db handles all core database interactions of the server
package db

import (
	"bytes"
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
}

// DatabaseThread is a template for writing new threads to the database
type DatabaseThread struct {
	ID                  uint64
	PostCtr, ImageCtr   uint32
	ReplyTime, BumpTime int64
	Subject, Board      string
	Log                 [][]byte
}

type executor interface {
	Exec(args ...interface{}) (sql.Result, error)
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

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, l := range l {
		if i != 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('{')
		buf.WriteString(strconv.FormatUint(l[0], 10))
		buf.WriteByte(',')
		buf.WriteString(strconv.FormatUint(l[1], 10))
		buf.WriteByte('}')
	}
	buf.WriteByte('}')

	return buf.String(), nil
}

// ValidateOP confirms the specified thread exists on specific board
func ValidateOP(id uint64, board string) (valid bool, err error) {
	err = prepared["validateOP"].QueryRow(id, board).Scan(&valid)
	if err == sql.ErrNoRows {
		return false, nil
	}
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
		name, trip, auth, img, imgName *string
		spoiler                        bool
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
		auth, img, imgName, linkRow(p.Links), linkRow(p.Backlinks), comm,
	)
	return err
}

func getExecutor(tx *sql.Tx, key string) executor {
	if tx != nil {
		return tx.Stmt(prepared[key])
	}
	return prepared[key]
}

// WriteThread writes a thread and it's OP to the database
func WriteThread(t DatabaseThread, p DatabasePost) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	log := pq.GenericArray{A: t.Log}
	_, err = tx.Stmt(prepared["writeOP"]).Exec(
		t.Board,
		log,
		t.ID,
		t.PostCtr,
		t.ImageCtr,
		t.ReplyTime,
		t.BumpTime,
		t.Subject,
	)
	if err != nil {
		tx.Rollback()
		return err
	}
	if err := WritePost(tx, p); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
