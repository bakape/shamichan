package db

import (
	"github.com/bakape/meguca/common"
	"github.com/dancannon/gorethink"
)

// IsLoggedIn check if the user is logged in with the specified session
func IsLoggedIn(user, session string) (bool, error) {
	if len(user) > common.MaxLenUserID || len(session) != common.LenSession {
		return false, common.ErrInvalidCreds
	}

	var loggedIn bool
	q := gorethink.
		Table("accounts").
		Get(user).
		Field("sessions").
		Field("token").
		Contains(session).
		Default(false)
	if err := One(q, &loggedIn); err != nil {
		return false, err
	}
	return loggedIn, nil
}
