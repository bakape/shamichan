package server

import (
	"database/sql"
	"errors"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	errInvalidCaptcha  = errors.New("invalid captcha")
	errInvalidPassword = errors.New("invalid password")
	errInvalidUserID   = errors.New("invalid login ID")
	errUserIDTaken     = errors.New("login ID already taken")
)

type loginCreds struct {
	ID, Password string
	auth.Captcha
}

type passwordChangeRequest struct {
	Old, New string
	auth.Captcha
}

// Register a new user account
func register(w http.ResponseWriter, r *http.Request) {
	var req loginCreds
	isValid := decodeJSON(w, r, &req) &&
		trimLoginID(&req.ID) &&
		validateUserID(w, req.ID) &&
		checkPasswordAndCaptcha(w, r, req.Password, req.Captcha)
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
	if err := db.WriteLoginSession(userID, token); err != nil {
		text500(w, r, err)
		return
	}

	// One hour less, so the cookie expires a bit before the DB session gets
	// deleted
	expires := time.Now().
		Add(time.Duration(config.Get().SessionExpiry)*time.Hour*24 - time.Hour)
	loginCookie := http.Cookie{
		Name:    "loginID",
		Value:   userID,
		Path:    "/",
		Expires: expires,
	}
	sessionCookie := http.Cookie{
		Name:    "session",
		Value:   token,
		Path:    "/",
		Expires: expires,
	}
	http.SetCookie(w, &loginCookie)
	http.SetCookie(w, &sessionCookie)
}

// Log into a registered user account
func login(w http.ResponseWriter, r *http.Request) {
	var req loginCreds
	switch {
	case !decodeJSON(w, r, &req):
		return
	case !trimLoginID(&req.ID):
		return
	case !auth.AuthenticateCaptcha(req.Captcha):
		text403(w, errInvalidCaptcha)
		return
	}

	hash, err := db.GetPassword(req.ID)
	switch err {
	case nil:
	case sql.ErrNoRows:
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
	commitLogout(w, r, func(req auth.SessionCreds) error {
		return db.LogOut(req.UserID, req.Session)
	})
}

// Common part of both logout endpoints
func commitLogout(
	w http.ResponseWriter,
	r *http.Request,
	fn func(auth.SessionCreds) error,
) {
	creds, ok := isLoggedIn(w, r)
	if !ok {
		return
	}

	if err := fn(creds); err != nil {
		text500(w, r, err)
	}
}

// Log out all sessions of the specific user
func logoutAll(w http.ResponseWriter, r *http.Request) {
	commitLogout(w, r, func(req auth.SessionCreds) error {
		return db.LogOutAll(req.UserID)
	})
}

// Change the account password
func changePassword(w http.ResponseWriter, r *http.Request) {
	var msg passwordChangeRequest
	if !decodeJSON(w, r, &msg) {
		return
	}
	creds, ok := isLoggedIn(w, r)
	if !ok || !checkPasswordAndCaptcha(w, r, msg.New, msg.Captcha) {
		return
	}

	// Get old hash
	hash, err := db.GetPassword(creds.UserID)
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
	if err := db.ChangePassword(creds.UserID, hash); err != nil {
		text500(w, r, err)
	}
}

// Check password length and authenticate captcha, if needed
func checkPasswordAndCaptcha(
	w http.ResponseWriter,
	r *http.Request,
	password string,
	captcha auth.Captcha,
) bool {
	switch {
	case password == "", len(password) > common.MaxLenPassword:
		text400(w, errInvalidPassword)
		return false
	case !auth.AuthenticateCaptcha(captcha):
		text403(w, errInvalidCaptcha)
		return false
	}
	return true
}

// Assert the user login session ID is valid and returns the login credentials
func isLoggedIn(w http.ResponseWriter, r *http.Request) (
	creds auth.SessionCreds, ok bool,
) {
	// Extract from cookies
	if c, err := r.Cookie("session"); err == nil {
		creds.Session = c.Value
	}
	if c, err := r.Cookie("loginID"); err == nil {
		creds.UserID = strings.TrimSpace(c.Value)
	}
	if creds.UserID == "" || creds.Session == "" {
		text403(w, errAccessDenied)
		return
	}

	ok, err := db.IsLoggedIn(creds.UserID, creds.Session)
	switch err {
	case common.ErrInvalidCreds:
		text403(w, err)
	case nil:
		if !ok {
			text403(w, errAccessDenied)
		}
	default:
		text500(w, r, err)
	}

	return
}

// Trim spaces from loginID. Chainable with other authenticators.
func trimLoginID(id *string) bool {
	*id = strings.TrimSpace(*id)
	return true
}
