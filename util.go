/*
 Contains various general helper functions
*/

package main

import (
	r "github.com/dancannon/gorethink"
	"strconv"
)

// throw panics, if there is an error. Rob Pike must never know.
func throw(err error) {
	if err != nil {
		panic(err)
	}
}

// checkAuth checks if the suplied Ident has enough or greater access right
// than requiered
func checkAuth(auth string, ident Ident) bool {
	return authRank(auth) <= authRank(ident.Auth)
}

// authRank determines the rank of the suplied authority class in the access
// level hierarchy
func authRank(auth string) int {
	for i, level := range [4]string{"dj", "janitor", "moderator", "admin"} {
		if auth == level {
			return i
		}
	}
	return -1
}

// rGet is a shorthand for executing RethinkDB queries and panicing on error.
func rGet(query r.Term) *r.Cursor {
	cursor, err := query.Run(rSession)
	throw(err)
	return cursor
}

// rExec executes a RethinkDB query and panics on error. To be used, when the
// returned status is unneeded and we want the goroutine to crash on error.
func rExec(query r.Term) {
	throw(query.Exec(rSession))
}

// shorthand for constructing thread queries
func getThread(id int) r.Term {
	return r.Table("threads").Get(id)
}

// shorthand for constructing post queries
func getPost(id, op int) r.Term {
    return getThread(op).Field("posts").Field(strconv.Itoa(id))
}
