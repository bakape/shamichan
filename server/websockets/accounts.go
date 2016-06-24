// Account and login -related message handlers

package websockets

import (
	"errors"

	"github.com/bakape/meguca/db"
	"golang.org/x/crypto/bcrypt"
)

// Account registration and login response codes
type accountResponse uint8

const (
	loginSuccess accountResponse = iota
	userNameTaken
	idTooShort
	idTooLong
	passwordTooShort
	passwordTooLong
)

type registrationRequest struct {
	ID       string `json:"id"`
	Password string `json:"password"`
}

// Register a new user account
func register(data []byte, c *Client) error {
	if c.loggedIn {
		return errors.New("already logged in")
	}

	var req registrationRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}

	code, err := handleRegistration(req.ID, req.Password)
	if err != nil {
		return err
	}
	if err := c.sendMessage(messageLogin, code); err != nil {
		return err
	}

	if code == loginSuccess {
		c.loggedIn = true
	}
	return nil
}

// Seperated into its own function for cleanliness and testability
func handleRegistration(id, password string) (
	code accountResponse, err error,
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
