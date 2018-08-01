package server

import (
	"database/sql"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	errInvalidCaptcha  = common.ErrAccessDenied("invalid captcha")
	errInvalidPassword = common.ErrInvalidInput("password")
	errInvalidUserID   = common.ErrInvalidInput("login ID")
	errUserIDTaken     = common.ErrInvalidInput("login ID already taken")
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
		validateUserID(w, r, req.ID) &&
		checkPasswordAndCaptcha(w, r, req.Password, req.Captcha)
	if !isValid {
		return
	}

	hash, err := auth.BcryptHash(req.Password, 10)
	if err != nil {
		httpError(w, r, err)
		return
	}

	// Check for collision and write to DB
	err = db.InTransaction(func(tx *sql.Tx) error {
		return db.RegisterAccount(tx, req.ID, hash)
	})
	switch err {
	case nil:
		commitLogin(w, r, req.ID)
	case db.ErrUserNameTaken:
		httpError(w, r, errUserIDTaken)
	default:
		httpError(w, r, err)
	}
}

// Separate function for easier chaining of validations
func validateUserID(w http.ResponseWriter, r *http.Request, id string) bool {
	if id == "" || len(id) > common.MaxLenUserID {
		httpError(w, r, errInvalidUserID)
		return false
	}
	return true
}

// If login successful, generate a session token and commit to DB. Otherwise
// write error message to client.
func commitLogin(w http.ResponseWriter, r *http.Request, userID string) {
	token, err := auth.RandomID(128)
	if err != nil {
		httpError(w, r, err)
		return
	}
	if err := db.WriteLoginSession(userID, token); err != nil {
		httpError(w, r, err)
		return
	}

	// One hour less, so the cookie expires a bit before the DB session gets
	// deleted
	expires := time.Now().
		Add(time.Duration(config.Get().SessionExpiry)*time.Hour*24 - time.Hour)
	http.SetCookie(w, &http.Cookie{
		Name:    "loginID",
		Value:   url.QueryEscape(userID),
		Path:    "/",
		Expires: expires,
	})
	http.SetCookie(w, &http.Cookie{
		Name:    "session",
		Value:   token,
		Path:    "/",
		Expires: expires,
	})
}

// Log into a registered user account
func login(w http.ResponseWriter, r *http.Request) {
	ip, err := auth.GetIP(r)
	if err != nil {
		httpError(w, r, err)
		return
	}
	var req loginCreds
	switch {
	case !decodeJSON(w, r, &req):
		return
	case !trimLoginID(&req.ID):
		return
	}
	err = db.AuthenticateCaptcha(req.Captcha, ip)
	if err != nil {
		httpError(w, r, errInvalidCaptcha)
		return
	}

	hash, err := db.GetPassword(req.ID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		httpError(w, r, common.ErrInvalidCreds)
		return
	default:
		httpError(w, r, err)
		return
	}

	switch err := auth.BcryptCompare(req.Password, hash); err {
	case nil:
		commitLogin(w, r, req.ID)
	case bcrypt.ErrMismatchedHashAndPassword:
		httpError(w, r, common.ErrInvalidCreds)
	default:
		httpError(w, r, err)
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
		httpError(w, r, err)
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
		httpError(w, r, err)
		return
	}

	// Validate old password
	switch err := auth.BcryptCompare(msg.Old, hash); err {
	case nil:
	case bcrypt.ErrMismatchedHashAndPassword:
		httpError(w, r, common.ErrInvalidCreds)
		return
	default:
		httpError(w, r, err)
		return
	}

	// Old password matched, write new hash to DB
	hash, err = auth.BcryptHash(msg.New, 10)
	if err != nil {
		httpError(w, r, err)
		return
	}
	if err := db.ChangePassword(creds.UserID, hash); err != nil {
		httpError(w, r, err)
	}
}

// Check password length and authenticate captcha, if needed
func checkPasswordAndCaptcha(
	w http.ResponseWriter,
	r *http.Request,
	password string,
	captcha auth.Captcha,
) bool {
	ip, err := auth.GetIP(r)
	if err != nil {
		httpError(w, r, err)
		return false
	}
	if password == "" || len(password) > common.MaxLenPassword {
		httpError(w, r, errInvalidPassword)
		return false
	}
	err = db.AuthenticateCaptcha(captcha, ip)
	if err != nil {
		httpError(w, r, errInvalidCaptcha)
		return false
	}
	return true
}

// Assert the user login session ID is valid and returns the login credentials
func isLoggedIn(w http.ResponseWriter, r *http.Request) (
	creds auth.SessionCreds, ok bool,
) {
	creds = extractLoginCreds(r)
	if creds.UserID == "" || creds.Session == "" {
		httpError(w, r, errAccessDenied)
		return
	}

	ok, err := db.IsLoggedIn(creds.UserID, creds.Session)
	switch err {
	case common.ErrInvalidCreds:
		httpError(w, r, err)
	case nil:
		if !ok {
			httpError(w, r, errAccessDenied)
		}
	default:
		httpError(w, r, err)
	}

	return
}

// Extract login credentials from cookies
func extractLoginCreds(r *http.Request) (creds auth.SessionCreds) {
	if c, err := r.Cookie("session"); err == nil {
		creds.Session = c.Value
	}
	if c, err := r.Cookie("loginID"); err == nil {
		creds.UserID, _ = url.QueryUnescape(strings.TrimSpace(c.Value))
	}
	return
}

// Trim spaces from loginID. Chainable with other authenticators.
func trimLoginID(id *string) bool {
	*id = strings.TrimSpace(*id)
	return true
}
