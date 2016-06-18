// Package auth determines and asserts client permissions to access and modify
// resources.
package auth

import (
	"github.com/bakape/meguca/config"
)

// Check checks if the suplied Ident is priveledged to perform the specified
// action
func Check(action string, ident Ident) bool {
	if class, ok := config.Get().Staff.Classes[ident.Auth]; ok {
		return class.Rights[action]
	}
	return false
}

// LookUpIdent determine access rights of an IP
func LookUpIdent(ip string) Ident {
	ident := Ident{IP: ip}

	// TODO: BANS

	return ident
}

// IsBoard confirms the string is a valid board
func IsBoard(board string) bool {
	if board == "all" {
		return true
	}
	for _, b := range config.Get().Boards.Enabled {
		if board == b {
			return true
		}
	}
	return false
}

// Ident is used to verify a client's access and write permissions
type Ident struct {
	Banned bool
	Auth   string // Indicates priveledged access rights for staff
	IP     string
}
