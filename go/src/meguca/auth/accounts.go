package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// SessionCreds is embed in every request that needs logged in authentication
type SessionCreds struct {
	UserID, Session string
}

// BcryptCompare compares a bcrypt hash with a user-supplied string
func BcryptCompare(password string, hash []byte) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(password))
}
