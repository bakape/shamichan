// Package db handles all core database interactions of the server
package db

import (
	"database/sql"

	"github.com/bakape/meguca/common"
	"github.com/lib/pq"
	"github.com/pquerna/ffjson/ffjson"
)

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
func WritePost(tx *sql.Tx, p common.DatabasePost) error {
	var ex executor
	if tx != nil {
		ex = tx.Stmt(prepared["writePost"])
	} else {
		ex = prepared["writePost"]
	}

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
		p.Editing, p.Deleted, spoiler, p.ID, p.OP, p.Time, p.Body, name, trip,
		auth, img, imgName, comm,
	)
	return err
}

// WriteThread writes a thread and it's OP to the database
func WriteThread(t common.DatabaseThread, p common.DatabasePost) error {
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
