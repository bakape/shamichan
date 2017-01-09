// Package db handles all core database interactions of the server
package db

import (
	"database/sql"

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
	ID                uint64
	PostCtr, ImageCtr uint32
	ReplyTime         int64
	Subject, Board    string
	Log               [][]byte
}

type executor interface {
	Exec(args ...interface{}) (sql.Result, error)
}

// // FindPost finds a post only by ID number
// func FindPost(id uint64) r.Term {
// 	return r.Table("posts").Get(id)
// }

// // ValidateOP confirms the specified thread exists on specific board
// func ValidateOP(id uint64, board string) (valid bool, err error) {
// 	err = One(FindThread(id).Field("board").Eq(board).Default(false), &valid)
// 	return
// }

// // FindThread is a  shorthand for retrieving a document from the "threads" table
// func FindThread(id uint64) r.Term {
// 	return r.Table("threads").Get(id)
// }

// // GetMain is a shorthand for retrieving a document from the "main" table
// func GetMain(id string) r.Term {
// 	return r.Table("main").Get(id)
// }

// // GetAccount is a shorthand for retrieving a document from the "accounts" table
// func GetAccount(id string) r.Term {
// 	return r.Table("accounts").Get(id)
// }

// // GetImage is a shorthand for retrieving a document from the "images" table
// func GetImage(id string) r.Term {
// 	return r.Table("images").Get(id)
// }

// // BoardCounter retrieves the history or "progress" counter of a board
// func BoardCounter(board string) (counter uint64, err error) {
// 	q := r.
// 		Table("posts").
// 		GetAllByIndex("board", board).
// 		Field("lastUpdated").
// 		Max().
// 		Default(0)
// 	err = One(q, &counter)
// 	return
// }

// // ThreadCounter retrieves the post counter of a thread to get a rough estimate
// // of the thread's progress
// func ThreadCounter(id uint64) (counter uint64, err error) {
// 	q := r.
// 		Table("posts").
// 		GetAllByIndex("op", id).
// 		Field("lastUpdated").
// 		Max().
// 		Default(0)
// 	err = One(q, &counter)
// 	return
// }

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
		auth, img, imgName, comm,
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

// WriteLinks writes a post's links to the database
func WriteLinks(tx *sql.Tx, src uint64, links common.LinkMap) error {
	return writeLinks(tx, "writeLinks", src, links)
}

// WriteBacklinks writes a post's backlinks to the database
func WriteBacklinks(tx *sql.Tx, src uint64, links common.LinkMap) error {
	return writeLinks(tx, "writeBacklinks", src, links)
}

func writeLinks(tx *sql.Tx, q string, src uint64, links common.LinkMap) error {
	ex := getExecutor(tx, q)
	for target, link := range links {
		_, err := ex.Exec(src, target, link.OP, link.Board)
		if err != nil {
			return err
		}
	}
	return nil
}
