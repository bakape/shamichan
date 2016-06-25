// Account and login -related message handlers

package websockets

import (
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
	"golang.org/x/crypto/bcrypt"
)

// Account registration and login response codes
type loginResponseCode uint8

const (
	loginSuccess loginResponseCode = iota
	userNameTaken
	wrongCredentials
	idTooShort
	idTooLong
	passwordTooShort
	passwordTooLong
)

var (
	errAlreadyLoggedIn = errInvalidMessage("already logged in")
)

// Request struct for logging in to an existing or registering a new account
type loginRequest struct {
	ID       string `json:"id"`
	Password string `json:"password"`
}

type loginResponse struct {
	Code    loginResponseCode `json:"code"`
	Session string            `json:"session"`
}

type authenticationRequest struct {
	ID      string `json:"id"`
	Session string `json:"session"`
}

// Register a new user account
func register(data []byte, c *Client) error {
	if c.loggedIn {
		return errAlreadyLoggedIn
	}

	var req loginRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}

	code, err := handleRegistration(req.ID, req.Password)
	if err != nil {
		return err
	}

	return commitLogin(code, req.ID, c)
}

// Seperated into its own function for cleanliness and testability
func handleRegistration(id, password string) (
	code loginResponseCode, err error,
) {
	// Validate string lengths
	switch {
	case len(id) < 3:
		code = idTooShort
	case len(id) > 20:
		code = idTooLong
	case len(password) < 6:
		code = passwordTooShort
	case len(password) > 30:
		code = passwordTooLong
	}
	if code > 0 {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(id+password), 10)
	if err != nil {
		return
	}

	// Check for collision and write to DB
	err = db.RegisterAccount(id, hash)
	switch err {
	case nil:
		code = loginSuccess
	case db.ErrUserNameTaken:
		code = userNameTaken
		err = nil
	}

	return
}

// If login succesful, generate a session token and comit to DB. Otherwise
// simply send the response code the client.
func commitLogin(code loginResponseCode, id string, c *Client) (err error) {
	msg := loginResponse{Code: code}
	if code == loginSuccess {
		msg.Session, err = util.RandomID(40)
		if err != nil {
			return err
		}

		query := db.GetAccount(id).Update(map[string]r.Term{
			"sessions": r.Row.Field("sessions").Append(msg.Session),
		})
		if err := db.Write(query); err != nil {
			return err
		}

		c.loggedIn = true
	}

	return c.sendMessage(messageLogin, msg)
}

// Log in a registered user account
func login(data []byte, c *Client) error {
	if c.loggedIn {
		return errAlreadyLoggedIn
	}

	var req loginRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}

	hash, err := db.GetLoginHash(req.ID)
	if err != nil {
		if err == r.ErrEmptyResult {
			return commitLogin(wrongCredentials, req.ID, c)
		}
		return err
	}

	var code loginResponseCode
	err = bcrypt.CompareHashAndPassword(hash, []byte(req.ID+req.Password))
	switch err {
	case bcrypt.ErrMismatchedHashAndPassword:
		code = wrongCredentials
	case nil:
		code = loginSuccess
	default:
		return err
	}

	return commitLogin(code, req.ID, c)
}

func authenticateSession(data []byte, c *Client) error {
	if c.loggedIn {
		return errAlreadyLoggedIn
	}

	var req authenticationRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}

	var isSession bool
	query := db.
		GetAccount(req.ID).
		Field("sessions").
		Contains(req.Session).
		Default(false)
	if err := db.One(query, &isSession); err != nil && err != r.ErrEmptyResult {
		return err
	}

	return c.sendMessage(messageAuthenticate, isSession)
}
