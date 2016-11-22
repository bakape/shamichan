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
	errInvalidCreds    = errors.New("invalid login credentials")
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

// Embed in every request that needs authentication
type sessionCreds struct {
	UserID, Session string
}

// type authenticationRequest struct {
// 	ID, Session string
// }

type passwordChangeRequest struct {
	sessionCreds
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
		text403(w, errInvalidCreds)
		return
	default:
		text500(w, r, err)
		return
	}

	switch err := auth.BcryptCompare(req.Password, hash); err {
	case nil:
		commitLogin(w, r, req.ID)
	case bcrypt.ErrMismatchedHashAndPassword:
		text403(w, errInvalidCreds)
	default:
		text500(w, r, err)
	}
}

// // Log out user from session and remove the session key from the database
// func logOut(_ []byte, c *Client) error {
// 	if !c.isLoggedIn() {
// 		return errNotLoggedIn
// 	}

// 	// Remove current session from user's session document
// 	query := db.GetAccount(c.UserID).
// 		Update(map[string]r.Term{
// 			"sessions": r.Row.
// 				Field("sessions").
// 				Filter(func(s r.Term) r.Term {
// 					return s.Field("token").Eq(c.sessionToken).Not()
// 				}),
// 		})
// 	return commitLogout(query, c)
// }

// // Common part of both logout functions
// func commitLogout(query r.Term, c *Client) error {
// 	c.UserID = ""
// 	c.sessionToken = ""
// 	if err := db.Write(query); err != nil {
// 		return err
// 	}

// 	return c.sendMessage(MessageLogout, true)
// }

// // Log out all sessions of the specific user
// func logOutAll(_ []byte, c *Client) error {
// 	if !c.isLoggedIn() {
// 		return errNotLoggedIn
// 	}

// 	query := db.GetAccount(c.UserID).
// 		Update(map[string][]string{
// 			"sessions": []string{},
// 		})
// 	return commitLogout(query, c)
// }

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
