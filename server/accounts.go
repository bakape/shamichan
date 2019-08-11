package server

import (
	"database/sql"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/jackc/pgx"
	"golang.org/x/crypto/bcrypt"
)

var (
	errInvalidCaptcha  = common.ErrAccessDenied("invalid captcha")
	errInvalidPassword = common.ErrInvalidInput("password")
	errInvalidUserID   = common.ErrInvalidInput("login ID")
	errUserIDTaken     = common.ErrInvalidInput("login ID already taken")
)

// TODO: Include captcha data in all applicable these post request

type loginCreds struct {
	ID, Password string
}

type passwordChangeRequest struct {
	Old, New string
}

// Register a new user account
func register(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		var req loginCreds
		err = decodeJSON(r, &req)
		if err != nil {
			return
		}
		trimLoginID(&req.ID)
		err = validateUserID(w, r, req.ID)
		if err != nil {
			return
		}
		err = checkPasswordAndCaptcha(w, r, req.Password)
		if err != nil {
			return
		}

		hash, err := auth.BcryptHash(req.Password, 10)
		if err != nil {
			return
		}

		// Check for collision and write to DB
		err = db.InTransaction(func(tx *pgx.Tx) error {
			return db.RegisterAccount(tx, req.ID, hash)
		})
		if err != nil {
			if err == db.ErrUserNameTaken {
				err = errUserIDTaken
			}
			return
		}
		return commitLogin(w, r, req.ID)
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Separate function for easier chaining of validations
func validateUserID(w http.ResponseWriter, r *http.Request, id string) error {
	if id == "" || len(id) > common.MaxLenUserID {
		return errInvalidUserID
	}
	return nil
}

// If login successful, generate a session token and commit to DB. Otherwise
// write error message to client.
func commitLogin(
	w http.ResponseWriter, r *http.Request,
	userID string,
) (err error) {
	token, err := auth.RandomID(128)
	if err != nil {
		return
	}
	err = db.WriteLoginSession(userID, token)
	if err != nil {
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
	return
}

// Log into a registered user account
func login(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		var req loginCreds
		err = decodeJSON(r, &req)
		if err != nil {
			return
		}
		trimLoginID(&req.ID)

		var session auth.Base64Token
		err = session.EnsureCookie(w, r)
		if err != nil {
			return
		}
		has, err := db.SolvedCaptchaRecently(session, time.Minute)
		if err != nil {
			return
		}
		if !has {
			err = errInvalidCaptcha
			return
		}

		hash, err := db.GetPassword(req.ID)
		switch err {
		case nil:
		case pgx.ErrNoRows:
			err = common.ErrInvalidCreds
			return
		default:
			return
		}

		err = auth.BcryptCompare(req.Password, hash)
		switch err {
		case nil:
		case bcrypt.ErrMismatchedHashAndPassword:
			err = common.ErrInvalidCreds
			return
		default:
			return
		}

		return commitLogin(w, r, req.ID)
	}()
	if err != nil {
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
	err := func() (err error) {
		creds, err := isLoggedIn(w, r)
		if err != nil {
			return
		}
		return fn(creds)
	}()
	if err != nil {
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
	err := func() (err error) {
		var msg passwordChangeRequest
		err = decodeJSON(r, &msg)
		if err != nil {
			return
		}

		creds, err := isLoggedIn(w, r)
		if err != nil {
			return
		}
		err = checkPasswordAndCaptcha(w, r, msg.New)
		if err != nil {
			return
		}

		// Get old hash
		hash, err := db.GetPassword(creds.UserID)
		if err != nil {
			return
		}

		// Validate old password
		err = auth.BcryptCompare(msg.Old, hash)
		switch err {
		case nil:
		case bcrypt.ErrMismatchedHashAndPassword:
			err = common.ErrInvalidCreds
			return
		default:
			return
		}

		// Old password matched, write new hash to DB
		hash, err = auth.BcryptHash(msg.New, 10)
		if err != nil {
			return
		}
		return db.ChangePassword(creds.UserID, hash)
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Check password length and authenticate captcha, if needed
func checkPasswordAndCaptcha(
	w http.ResponseWriter, r *http.Request,
	password string,
) (
	err error,
) {
	if password == "" || len(password) > common.MaxLenPassword {
		return errInvalidPassword
	}
	var session auth.Base64Token
	err = session.EnsureCookie(w, r)
	if err != nil {
		return
	}
	has, err := db.SolvedCaptchaRecently(session, time.Minute)
	if err != nil {
		return
	}
	if !has {
		err = errInvalidCaptcha
	}
	return
}

// Assert the user login session ID is valid and returns the login credentials
func isLoggedIn(w http.ResponseWriter, r *http.Request,
) (
	creds auth.SessionCreds, err error,
) {
	creds = extractLoginCreds(r)
	if creds.UserID == "" || creds.Session == "" {
		err = errAccessDenied
		return
	}

	ok, err := db.IsLoggedIn(creds.UserID, creds.Session)
	if err != nil {
		return
	}
	if !ok {
		err = errAccessDenied
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

// Trim spaces from loginID
func trimLoginID(id *string) {
	*id = strings.TrimSpace(*id)
}
