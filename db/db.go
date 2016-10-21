// Package db handles all core database intercations of the server
package db

import (
	"errors"

	"github.com/bakape/meguca/auth"
	r "github.com/dancannon/gorethink"
)

var (
	// ErrUserNameTaken denotes a user name the client is trying  to register
	// with is already taken
	ErrUserNameTaken = errors.New("user name already taken")
)

var postReservationQuery = GetMain("info").
	Update(
		map[string]r.Term{
			"postCtr": r.Row.Field("postCtr").Add(1),
		},
		r.UpdateOpts{
			ReturnChanges: true,
		},
	).
	Field("changes").
	AtIndex(0).
	Field("new_val").
	Field("postCtr")

// Exec excutes the query and only returns an error, if any. Do not use for
// write queries.
func Exec(query r.Term) error {
	return query.Exec(RSession)
}

// Write executes the inner query and returns an error, if any. Only use this
// function for write queries
func Write(query r.Term) error {
	_, err := query.RunWrite(RSession)
	return err
}

// WriteAll executes passed write queries in order. Returns on first error.
func WriteAll(qs []r.Term) error {
	for _, q := range qs {
		if err := Write(q); err != nil {
			return err
		}
	}
	return nil
}

// One writes the query result into the target pointer or throws an error
func One(query r.Term, res interface{}) error {
	c, err := query.Run(RSession)
	if err != nil {
		return err
	}
	return c.One(res)
}

// All writes all responses into target pointer to slice or returns error
func All(query r.Term, res interface{}) error {
	c, err := query.Run(RSession)
	if err != nil {
		return err
	}
	return c.All(res)
}

// FindPost finds a post only by ID number
func FindPost(id int64) r.Term {
	return r.Table("posts").Get(id)
}

// ValidateOP confirms the specified thread exists on specific board
func ValidateOP(id int64, board string) (valid bool, err error) {
	err = One(FindThread(id).Field("board").Eq(board).Default(false), &valid)
	return
}

// FindThread is a  shorthand for retrieving a document from the "threads" table
func FindThread(id int64) r.Term {
	return r.Table("threads").Get(id)
}

// GetMain is a shorthand for retrieving a document from the "main" table
func GetMain(id string) r.Term {
	return r.Table("main").Get(id)
}

// GetAccount is a shorthand for retrieving a document from the "accounts" table
func GetAccount(id string) r.Term {
	return r.Table("accounts").Get(id)
}

// GetImage is a shorthand for retrieving a document from the "images" table
func GetImage(id string) r.Term {
	return r.Table("images").Get(id)
}

// Insert is a shorthand for inserting documents or slices of documents into a
// table
func Insert(table string, doc interface{}) error {
	return Write(r.Table(table).Insert(doc))
}

// PostCounter retrieves the current global post count
func PostCounter() (counter int64, err error) {
	err = One(GetMain("info").Field("postCtr"), &counter)
	return
}

// BoardCounter retrieves the history or "progress" counter of a board
func BoardCounter(board string) (counter int64, err error) {
	err = One(GetMain("boardCtrs").Field(board).Default(0), &counter)
	return
}

// ThreadCounter retrieves the post counter of a thread to get a rough estimate
// of the thread's progress
func ThreadCounter(id int64) (counter int64, err error) {
	err = One(FindThread(id).Field("postCtr"), &counter)
	return
}

// RegisterAccount writes the ID and password hash of a new user account to the
// database
func RegisterAccount(ID string, hash []byte) error {
	err := Insert("accounts", auth.User{
		ID:       ID,
		Password: hash,
	})
	if r.IsConflictErr(err) {
		return ErrUserNameTaken
	}
	return err
}

// GetLoginHash retrieves the login hash of the registered user account
func GetLoginHash(id string) (hash []byte, err error) {
	query := GetAccount(id).Field("password").Default(nil)
	err = One(query, &hash)
	return
}

// ReservePostID reserves a post ID number for post and thread creation
func ReservePostID() (id int64, err error) {
	err = One(postReservationQuery, &id)
	return
}

// IncrementBoardCounter increments the progress counter of a board by 1. To be
// used on post and thread creation
func IncrementBoardCounter(board string) error {
	q := GetMain("boardCtrs").
		Update(map[string]r.Term{
			board: r.Row.Field(board).Default(0).Add(1),
		})
	return Write(q)
}
