/*
 Common database functions and helper types
*/

package server

import (
	r "github.com/dancannon/gorethink"
)

var db func() Database

// Database eases writing test by providing an interface for mock-databases to
// implement
type Database interface {
	Do(r.Term) Database
	Exec()
	One(interface{})
	All(interface{})
}

// DatabaseHelper simplifies managing queries, by providing extra utility
type DatabaseHelper struct {
	query r.Term
}

// Do is a chainable method for defining the gorethink query to run
func (d DatabaseHelper) Do(query r.Term) Database {
	d.query = query
	return d
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
	db().Do(getPost(id).Field("op")).One(&op)
	return
}

// parentBoard determines the parent board of the post
func parentBoard(id uint64) (board string) {
	db().Do(getPost(id).Field("board")).One(&board)
	return
}

// ValidateOP confirms the specified thread exists on specific board
func validateOP(id uint64, board string) bool {
	return parentBoard(id) == board && parentThread(id) == id
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
	db().Do(r.Table("main").Get("info").Field("postCtr")).One(&counter)
	return
}

// Retrieve the history or "progress" counter of a board
func boardCounter(board string) (counter uint64) {
	db().
		Do(r.Table("main").Get("info").Field("postCtr").Default(0)).
		One(&counter)
	return
}

// Retrieve the history or "progress" counter of a thread
func threadCounter(id uint64) (counter uint64) {
	db().Do(r.Table("posts").GetAllByIndex("op", id).Count()).One(&counter)
	return
}
