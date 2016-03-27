// Package db handles all database intercations of the server
package db

import (
	"fmt"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

// DatabaseHelper simplifies managing queries, by providing extra utility
type DatabaseHelper struct {
	query r.Term
}

// Exec excutes the inner query and only returns an error, if any
func (d DatabaseHelper) Exec() error {
	return d.query.Exec(RSession)
}

// One writes the query result into the target pointer or throws an error
func (d DatabaseHelper) One(res interface{}) error {
	c, err := d.query.Run(RSession)
	if err != nil {
		return err
	}
	c.One(res)
	return nil
}

// All writes all responses into target pointer to slice or returns error
func (d DatabaseHelper) All(res interface{}) error {
	c, err := d.query.Run(RSession)
	if err != nil {
		return err
	}
	c.All(res)
	return nil
}

// parentThread determines the parent thread of a post
func parentThread(id uint64) (op uint64, err error) {
	err = DB()(getPost(id).Field("op").Default(0)).One(&op)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving parent thread number: %d", id)
		err = util.WrapError(msg, err)
	}
	return
}

// parentBoard determines the parent board of the post
func parentBoard(id uint64) (board string, err error) {
	err = DB()(getPost(id).Field("board").Default("")).One(&board)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving parent board: %d", id)
		err = util.WrapError(msg, err)
	}
	return
}

// ValidateOP confirms the specified thread exists on specific board
func ValidateOP(id uint64, board string) (valid bool, err error) {
	var b string
	err = DB()(getThread(id).Field("board").Default("")).One(&b)
	if err != nil {
		msg := fmt.Sprintf("Error validating OP: %d of board %s", id, board)
		err = util.WrapError(msg, err)
	}
	valid = b == board
	return
}

// shorthand for constructing thread queries
func getThread(id uint64) r.Term {
	return r.Table("threads").Get(id)
}

// shorthand for constructing post queries
func getPost(id uint64) r.Term {
	return r.Table("posts").Get(id)
}

// PostCounter retrieves the current post counter number
func PostCounter() (counter uint64, err error) {
	err = DB()(r.Table("main").Get("info").Field("postCtr")).One(&counter)
	if err != nil {
		err = util.WrapError("Error retrieving post counter", err)
	}
	return
}

// BoardCounter retrieves the history or "progress" counter of a board
func BoardCounter(board string) (counter uint64, err error) {
	err = DB()(r.Table("main").Get("histCounts").Field(board).Default(0)).
		One(&counter)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving board counter: %s", board)
		err = util.WrapError(msg, err)
	}
	return
}

// ThreadCounter retrieve the history or "progress" counter of a thread
func ThreadCounter(id uint64) (counter uint64, err error) {
	err = DB()(r.Table("posts").GetAllByIndex("op", id).Count().Sub(1)).
		One(&counter)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving thread counter: %d", id)
		err = util.WrapError(msg, err)
	}
	return
}
