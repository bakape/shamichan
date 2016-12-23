package server

import (
	"errors"
	"net/http"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/dancannon/gorethink"
	"golang.org/x/crypto/bcrypt"
)

var (
	errInvalidCaptcha  = errors.New("invalid captcha")
	errInvalidPassword = errors.New("invalid password")
	errInvalidUserID   = errors.New("invalid login ID")
	errUserIDTaken     = errors.New("login ID already taken")
)

// Request struct for logging in to an existing or registering a new account
type loginRequest struct {
	common.Captcha
	loginCreds
}

type loginCreds struct {
	ID, Password string
}

type passwordChangeRequest struct {
	auth.SessionCreds
	common.Captcha
	Old, New string
}

// Register a new user account
func register(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	isValid := decodeJSON(w, r, &req) &&
		validateUserID(w, req.ID) &&
		checkPasswordAndCaptcha(w, r, req.Password, auth.GetIP(r), req.Captcha)
	if !isValid {
		return
	}

	hash, err := auth.BcryptHash(req.Password, 10)
	if err != nil {
		text500(w, r, err)
	}

	// Check for collision and write to DB
	switch err := db.RegisterAccount(req.ID, hash); err {
	case nil:
	case db.ErrUserNameTaken:
		text400(w, errUserIDTaken)
		return
	default:
		text500(w, r, err)
		return
	}

	commitLogin(w, r, req.ID)
}

// Separate function for easier chaining of validations
func validateUserID(w http.ResponseWriter, id string) bool {
	if id == "" || len(id) > common.MaxLenUserID {
		text400(w, errInvalidUserID)
		return false
	}
	return true
}

// If login successful, generate a session token and commit to DB. Otherwise
// write error message to client.
func commitLogin(w http.ResponseWriter, r *http.Request, userID string) {
	token, err := auth.RandomID(128)
	if err != nil {
		text500(w, r, err)
		return
	}

	expiryTime := time.Duration(config.Get().SessionExpiry) * time.Hour * 24
	session := auth.Session{
		Token:   token,
		Expires: time.Now().Add(expiryTime),
	}
	q := db.GetAccount(userID).Update(map[string]gorethink.Term{
		"sessions": gorethink.Row.Field("sessions").Append(session),
	})
	if err := db.Write(q); err != nil {
		text500(w, r, err)
		return
	}

	w.Write([]byte(token))
}

// Log into a registered user account
func login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	switch {
	case !decodeJSON(w, r, &req):
		return
	case !auth.AuthenticateCaptcha(req.Captcha, auth.GetIP(r)):
		text403(w, errInvalidCaptcha)
		return
	}

	hash, err := db.GetLoginHash(req.ID)
	switch err {
	case nil:
	case gorethink.ErrEmptyResult:
		text403(w, common.ErrInvalidCreds)
		return
	default:
		text500(w, r, err)
		return
	}

	switch err := auth.BcryptCompare(req.Password, hash); err {
	case nil:
		commitLogin(w, r, req.ID)
	case bcrypt.ErrMismatchedHashAndPassword:
		text403(w, common.ErrInvalidCreds)
	default:
		text500(w, r, err)
	}
}

// Log out user from session and remove the session key from the database
func logout(w http.ResponseWriter, r *http.Request) {
	commitLogout(w, r, func(req auth.SessionCreds) gorethink.Term {
		// Remove current session from user's session document
		return db.GetAccount(req.UserID).Update(map[string]gorethink.Term{
			"sessions": gorethink.Row.
				Field("sessions").
				Filter(func(s gorethink.Term) gorethink.Term {
					return s.Field("token").Eq(req.Session).Not()
				}),
		})
	})
}

// Common part of both logout endpoints
func commitLogout(
	w http.ResponseWriter,
	r *http.Request,
	fn func(auth.SessionCreds) gorethink.Term,
) {
	var req auth.SessionCreds
	if !decodeJSON(w, r, &req) || !isLoggedIn(w, r, req.UserID, req.Session) {
		return
	}

	if err := db.Write(fn(req)); err != nil {
		text500(w, r, err)
	}
}

// Log out all sessions of the specific user
func logoutAll(w http.ResponseWriter, r *http.Request) {
	commitLogout(w, r, func(req auth.SessionCreds) gorethink.Term {
		return db.GetAccount(req.UserID).Update(map[string][]string{
			"sessions": []string{},
		})
	})
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
		text403(w, common.ErrInvalidCreds)
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
	isValid, err := db.IsLoggedIn(user, session)
	switch err {
	case common.ErrInvalidCreds:
		text403(w, err)
		return false
	case nil:
	default:
		text500(w, r, err)
		return false
	}

	if !isValid {
		text403(w, common.ErrInvalidCreds)
		return false
	}

	return true
}
