package db

import (
	"errors"

	"github.com/bakape/meguca/common"
	"github.com/lib/pq"
)

var (
	// ErrUserNameTaken denotes a user name the client is trying  to register
	// with is already taken
	ErrUserNameTaken = errors.New("user name already taken")
)

func init() {
	AddPrepared(
		"isLoggedIn",
		`SELECT EXISTS (
			SELECT true FROM sessions
				WHERE account = $1 AND token = $2
		);`,
	)
}

// IsLoggedIn check if the user is logged in with the specified session
func IsLoggedIn(user, session string) (bool, error) {
	if len(user) > common.MaxLenUserID || len(session) != common.LenSession {
		return false, common.ErrInvalidCreds
	}

	var loggedIn bool
	err := Prepared["isLoggedIn"].QueryRow(user, session).Scan(&loggedIn)
	if err != nil {
		return false, err
	}
	return true, nil
}

// RegisterAccount writes the ID and password hash of a new user account to the
// database
func RegisterAccount(ID string, hash []byte) error {
	_, err := DB.Exec(
		`INSERT INTO accounts (id, password) VALUES ($1, $2)`,
		ID, hash,
	)
	if err, ok := err.(*pq.Error); ok && err.Code.Name() == "unique_violation" {
		return ErrUserNameTaken
	}
	return err
}
