// Package db handles all database intercations of the server
package db

import (
	"errors"
	"fmt"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

var (
	// Precompiled query for extracting only the changed fields from the replication
	// log feed
	formatUpdateFeed = r.Row.
				Field("new_val").
				Field("log").
				Slice(
			r.Row.
				Field("old_val").
				Field("log").
				Count().
				Default(0),
		)

	// ErrUserNameTaken denotes a user name the client is trying  to register
	// with is already taken
	ErrUserNameTaken = errors.New("user name already taken")
)

// DatabaseHelper simplifies managing queries, by providing extra utility
type DatabaseHelper struct {
	query r.Term
}

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

// ParentThread determines the parent thread of a post. Returns 0, if post not
// found.
func ParentThread(id int64) (op int64, err error) {
	query := r.
		Table("threads").
		Filter(r.Row.Field("posts").HasFields(util.IDToString(id))).
		Field("id").
		Default(0)
	err = One(query, &op)
	if err != nil && err != r.ErrEmptyResult {
		msg := fmt.Sprintf("error retrieving parent thread: %d", id)
		err = util.WrapError(msg, err)
	}
	return
}

// ValidateOP confirms the specified thread exists on specific board
func ValidateOP(id int64, board string) (valid bool, err error) {
	err = One(getThread(id).Field("board").Eq(board).Default(false), &valid)
	if err != nil {
		msg := fmt.Sprintf("error validating OP %d of board %s", id, board)
		err = util.WrapError(msg, err)
	}
	return
}

// shorthand for retrieving a document from the "threads" table
func getThread(id int64) r.Term {
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

// PostCounter retrieves the current global post count
func PostCounter() (counter int64, err error) {
	err = One(GetMain("info").Field("postCtr"), &counter)
	if err != nil {
		err = util.WrapError("error retrieving post counter", err)
	}
	return
}

// BoardCounter retrieves the history or "progress" counter of a board
func BoardCounter(board string) (counter int64, err error) {
	err = One(GetMain("histCounts").Field(board).Default(0), &counter)
	if err != nil {
		msg := fmt.Sprintf("error retrieving board counter: %s", board)
		err = util.WrapError(msg, err)
	}
	return
}

// ThreadCounter retrieve the history or "progress" counter of a thread
func ThreadCounter(id int64) (counter int64, err error) {
	err = One(getThread(id).Field("log").Count(), &counter)
	if err != nil {
		msg := fmt.Sprintf("error retrieving thread counter: %d", id)
		err = util.WrapError(msg, err)
	}
	return
}

// StreamUpdates produces a stream of the replication log updates for the
// specified thread and sends it on read. Close the close channel to stop
// receiving updates. The intial contents of the log are returned immediately.
func StreamUpdates(
	id int64,
	write chan<- []byte,
	closer *util.AtomicCloser,
) ([][]byte, error) {
	cursor, err := getThread(id).
		Changes(r.ChangesOpts{IncludeInitial: true}).
		Map(formatUpdateFeed).
		Run(RSession)
	if err != nil {
		return nil, util.WrapError("error establishing update feed", err)
	}

	read := make(chan [][]byte)
	cursor.Listen(read)
	initial := <-read

	go func() {
		for closer.IsOpen() {
			// Several update messages may come from the feed at a time.
			// Separate and send each individually.
			messageStack := <-read
			for _, msg := range messageStack {
				write <- msg
			}
		}
	}()

	return initial, nil
}

// RegisterAccount writes the ID and password hash of a new user account to the
// database
func RegisterAccount(ID string, hash []byte) error {
	user := auth.User{
		ID:       ID,
		Password: hash,
	}
	err := Write(r.Table("accounts").Insert(user))
	if r.IsConflictErr(err) {
		return ErrUserNameTaken
	}
	return err
}
