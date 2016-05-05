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

// ParentThread determines the parent thread of a post. Returns 0, if post not
// found.
func ParentThread(id int64) (op int64, err error) {
	query := r.
		Table("threads").
		Filter(r.Row.Field("posts").HasFields(util.IDToString(id))).
		Field("id").
		Default(0)
	err = DB(query).One(&op)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving parent thread number: %d", id)
		err = util.WrapError(msg, err)
	}
	return
}

// ValidateOP confirms the specified thread exists on specific board
func ValidateOP(id int64, board string) (valid bool, err error) {
	err = DB(getThread(id).Field("board").Eq(board).Default(false)).One(&valid)
	if err != nil {
		msg := fmt.Sprintf("Error validating OP: %d of board %s", id, board)
		err = util.WrapError(msg, err)
	}
	return
}

// shorthand for constructing thread queries
func getThread(id int64) r.Term {
	return r.Table("threads").Get(id)
}

// shorthand for constructing post queries
func getPost(id, op int64) r.Term {
	return getThread(id).Field("posts").Field(id)
}

// GetMain is a shorthand for constructing main table queries
func GetMain(id string) r.Term {
	return r.Table("main").Get(id)
}

// PostCounter retrieves the current global post count
func PostCounter() (counter int64, err error) {
	err = DB(GetMain("info").Field("postCtr")).One(&counter)
	if err != nil {
		err = util.WrapError("Error retrieving post counter", err)
	}
	return
}

// BoardCounter retrieves the history or "progress" counter of a board
func BoardCounter(board string) (counter int64, err error) {
	err = DB(GetMain("histCounts").Field(board).Default(0)).One(&counter)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving board counter: %s", board)
		err = util.WrapError(msg, err)
	}
	return
}

// ThreadCounter retrieve the history or "progress" counter of a thread
func ThreadCounter(id int64) (counter int64, err error) {
	err = DB(getThread(id).Field("log").Count()).One(&counter)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving thread counter: %d", id)
		err = util.WrapError(msg, err)
	}
	return
}

// ReplicationLog retrieves the replication log of the specified thread
func ReplicationLog(id int64) (log [][]byte, err error) {
	err = DB(getThread(id).Field("log")).All(&log)
	return
}
