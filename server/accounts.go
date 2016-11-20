package server

import (
	"errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	"github.com/dancannon/gorethink"
)

var (
	errInvalidPassword = errors.New("invalid password")
)

type passwordChangeRequest struct {
	loginCredentials
	common.Captcha
	Old, New string
}

// Change the account password
func changePassword(w http.ResponseWriter, r *http.Request) {
	var msg passwordChangeRequest
	isValid := decodeJSON(w, r, &msg) &&
		isLoggedIn(w, r, msg.UserID, msg.Session) &&
		checkPasswordAndCaptcha(w, r, msg.New, auth.GetIP(r), msg.Captcha)
	if !isValid {
		return
	}

	// Get old hash
	hash, err := db.GetLoginHash(msg.UserID)
	if err != nil {
		text500(w, r, err)
		return
	}

	// Validate old password
	switch err := auth.BcryptCompare(msg.Old, hash); err {
	case nil:
	case bcrypt.ErrMismatchedHashAndPassword:
		text403(w, errInvalidCreds)
		return
	default:
		text500(w, r, err)
		return
	}

	// Old password matched, write new hash to DB
	hash, err = auth.BcryptHash(msg.New, 10)
	if err != nil {
		text500(w, r, err)
		return
	}

	q := db.GetAccount(msg.UserID).
		Update(map[string][]byte{
			"password": hash,
		})
	if err := db.Write(q); err != nil {
		text500(w, r, err)
		return
	}
}

// Check password length and authenticate captcha, if needed
func checkPasswordAndCaptcha(
	w http.ResponseWriter,
	r *http.Request,
	password, ip string,
	captcha common.Captcha,
) bool {
	switch {
	case password == "", len(password) > common.MaxLenPassword:
		text400(w, errInvalidPassword)
		return false
	case !auth.AuthenticateCaptcha(captcha, ip):
		text403(w, errInvalidCaptcha)
		return false
	}
	return true
}

// Assert the user login session ID is valid
func isLoggedIn(
	w http.ResponseWriter,
	r *http.Request,
	user, session string,
) bool {
	if len(user) > common.MaxLenUserID || len(session) != common.LenSession {
		text403(w, errInvalidCreds)
		return false
	}

	var isValid bool
	q := gorethink.
		Table("accounts").
		Get(user).
		Field("sessions").
		Map(func(session gorethink.Term) gorethink.Term {
			return session.Field("token")
		}).
		Contains(session).
		Default(false)
	if err := db.One(q, &isValid); err != nil {
		text500(w, r, err)
		return false
	}

	if !isValid {
		text403(w, errInvalidCreds)
		return false
	}

	return true
}
