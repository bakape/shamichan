// Package auth determines and asserts client permissions to access and modify
// resources.
package auth

import (
	"github.com/bakape/meguca/config"
)

// Check checks if the suplied Ident is priveledged to perform the specified
// action
func Check(action string, ident Ident) bool {
	if class, ok := config.Config.Staff.Classes[ident.Auth]; ok {
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

// CanAccessBoard confirms the client has rights to access the board
func CanAccessBoard(board string, ident Ident) bool {
	var isBoard bool
	if board == "all" {
		isBoard = true
	} else {
		for _, b := range config.Config.Boards.Enabled {
			if board == b {
				isBoard = true
				break
			}
		}
	}
	return isBoard && !ident.Banned
}

// Ident is used to verify a client's access and write permissions
type Ident struct {
	Banned bool
	Auth   string // Indicates priveledged access rights for staff
	IP     string
}
