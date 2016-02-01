/*
 Common database functions and helper types
*/

package server

import (
	r "github.com/dancannon/gorethink"
)

// DatabaseHelper simplifies managing queries, by providing extra utility
type DatabaseHelper struct {
	query r.Term
}

// Exec excutes the inner query and only returns an error, if any
func (d DatabaseHelper) Exec() {
	err := d.query.Exec(rSession)
	throw(err)
}

// One writes the query result into the target pointer or returns error
func (d DatabaseHelper) One(res interface{}) {
	c, err := d.query.Run(rSession)
	throw(err)
	c.One(res)
}

// All writes all responses into target pointer to slice or returns error
func (d DatabaseHelper) All(res interface{}) {
	c, err := d.query.Run(rSession)
	throw(err)
	c.All(res)
}

// parentThread determines the parent thread of a post
func parentThread(id uint64) (op uint64) {
	db()(getPost(id).Field("op").Default(0)).One(&op)
	return
}

// parentBoard determines the parent board of the post
func parentBoard(id uint64) (board string) {
	db()(getPost(id).Field("board").Default("")).One(&board)
	return
}

// ValidateOP confirms the specified thread exists on specific board
func validateOP(id uint64, board string) bool {
	var b string
	db()(getThread(id).Field("board").Default("")).One(&b)
	return b == board
}

// shorthand for constructing thread queries
func getThread(id uint64) r.Term {
	return r.Table("threads").Get(id)
}

// shorthand for constructing post queries
func getPost(id uint64) r.Term {
	return r.Table("posts").Get(id)
}

// Retrieve the current post counter number
func postCounter() (counter uint64) {
	db()(r.Table("main").Get("info").Field("postCtr")).One(&counter)
	return
}

// Retrieve the history or "progress" counter of a board
func boardCounter(board string) (counter uint64) {
	db()(r.Table("main").Get("histCounts").Field(board).Default(0)).
		One(&counter)
	return
}

// Retrieve the history or "progress" counter of a thread
func threadCounter(id uint64) (counter uint64) {
	db()(r.Table("posts").GetAllByIndex("op", id).Count().Sub(1)).One(&counter)
	return
}
