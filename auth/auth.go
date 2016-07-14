// Package auth determines and asserts client permissions to access and modify
// resources.
package auth

import (
	"net/url"
	"time"

	"github.com/bakape/meguca/config"
)

// User contains ID, password hash and board-related data of a registered user
// account
type User struct {
	ID       string    `gorethink:"id"`
	Password []byte    `gorethink:"password"`
	Sessions []Session `gorethink:"sessions"`
}

// Session contains the token and expiry time of a single authenticated login
// session
type Session struct {
	Token   string    `gorethink:"token"`
	Expires time.Time `gorethink:"expires"`
}

// Ident is used to verify a client's access and write permissions
type Ident struct {
	User
	IP string
}

// LookUpIdent determine access rights of an IP
func LookUpIdent(ip string) Ident {
	ident := Ident{IP: ip}

	// TODO: Bans and Authorisation

	return ident
}

// IsBoard confirms the string is a valid board
func IsBoard(board string) bool {
	if board == "all" {
		return true
	}
	return IsNonMetaBoard(board)
}

// IsNonMetaBoard returns wheather a valid board is a classic board and not
// some other path that emulates a board
func IsNonMetaBoard(board string) bool {
	board, _ = url.QueryUnescape(board) // For non-alphanumeric boards
	for _, b := range config.Get().Boards {
		if board == b {
			return true
		}
	}
	return false
}
