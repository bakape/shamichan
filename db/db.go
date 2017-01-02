// Package db handles all core database interactions of the server
package db

import r "github.com/dancannon/gorethink"

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

// Exec executes the query and only returns an error, if any. Do not use for
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
func WriteAll(qs ...r.Term) error {
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
func FindPost(id uint64) r.Term {
	return r.Table("posts").Get(id)
}

// ValidateOP confirms the specified thread exists on specific board
func ValidateOP(id uint64, board string) (valid bool, err error) {
	err = One(FindThread(id).Field("board").Eq(board).Default(false), &valid)
	return
}

// FindThread is a  shorthand for retrieving a document from the "threads" table
func FindThread(id uint64) r.Term {
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

// BoardCounter retrieves the history or "progress" counter of a board
func BoardCounter(board string) (counter uint64, err error) {
	q := r.
		Table("posts").
		GetAllByIndex("board", board).
		Field("lastUpdated").
		Max().
		Default(0)
	err = One(q, &counter)
	return
}

// ThreadCounter retrieves the post counter of a thread to get a rough estimate
// of the thread's progress
func ThreadCounter(id uint64) (counter uint64, err error) {
	q := r.
		Table("posts").
		GetAllByIndex("op", id).
		Field("lastUpdated").
		Max().
		Default(0)
	err = One(q, &counter)
	return
}

// ReservePostID reserves a post ID number for post and thread creation
func ReservePostID() (id uint64, err error) {
	err = One(postReservationQuery, &id)
	return
}
