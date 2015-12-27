/*
 Contains various general helper functions
*/

package main

import (
	"crypto/md5"
	"encoding/hex"
	r "github.com/dancannon/gorethink"
	"strconv"
)

// throw panics, if there is an error. Rob Pike must never know.
func throw(err error) {
	if err != nil {
		panic(err)
	}
}

// checkAuth checks if the suplied Ident is priveledged to perform an action
func checkAuth(action string, ident Ident) bool {
	if class, ok := config.Staff.Classes[ident.Auth]; ok {
		return class.Rights[action]
	}
	return false
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

// Determine access rights of an IP
func lookUpIdent(ip string) Ident {
	ident := Ident{IP: ip}

	// TODO: BANS

	return ident
}

// Confirm client has rights to access board
func canAccessBoard(board string, ident Ident) bool {
	if board == config.Boards.Staff && !checkAuth("accessStaffBoard", ident) {
		return false
	}
	_, ok := config.Boards.Boards[board]
	return !ident.Banned && ok
}

// Confirm thread exists and client has rights to access it's board
func canAccessThread(id int, board string, ident Ident) bool {
	if !canAccessBoard(board, ident) {
		return false
	}
	var deleted bool
	rGet(getThread(id).Field("deleted").Default(false)).One(&deleted)
	if deleted && !checkAuth("seeModeration", ident) {
		return false
	}
	return true
}

// Compute a truncated MD5 hash from a buffer
func hashBuffer(buf []byte) string {
	hasher := md5.New()
	hasher.Write(buf)
	return hex.EncodeToString(hasher.Sum(nil))[16:]
}
