// Wecbsocket message handlers

package websockets

import (
	"encoding/json"
	"errors"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
	"golang.org/x/crypto/bcrypt"
)

// identifier codes for websocket message types
type messageType uint8

// 1 - 29 modify post model state
const (
	messageInvalid messageType = iota
	messageInsertThread
	messageInsertPost
)

// >= 30 are miscelenious and do not write to post models
const (
	// Update feeds
	messageSynchronise messageType = 30 + iota
	messageResynchronise
	messageSwitchSync

	// Account management
	messageRegister
	messageLogin
	messageLogout
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

// Lookup table for message handlers
var handlers = map[messageType]func([]byte, *Client) error{
	messageSynchronise:   synchronise,
	messageResynchronise: resynchronise,
	messageRegister:      register,
}

// Error while parsing the message. Denotes that either the message does not
// follow the structural spec or contains optional fields in unsupported
// combinations.
type errInvalidMessage string

func (e errInvalidMessage) Error() string {
	return string(e)
}

var (
	errInvalidStructure = errInvalidMessage("Invalid message structure")
	errInvalidBoard     = errInvalidMessage("Invalid board")
	errInvalidThread    = errInvalidMessage("Invalid thread")
	errInvalidCounter   = errInvalidMessage("Invalid progress counter")
)

// Decode message JSON into the suplied type
func decodeMessage(data []byte, dest interface{}) error {
	err := json.Unmarshal(data, dest)
	if err != nil {
		return errInvalidStructure
	}
	return nil
}

type syncMessage struct {
	Ctr    int64  `json:"ctr"`
	Thread int64  `json:"thread"`
	Board  string `json:"board"`
}

// Syncronise the client to a certain thread, assign it's ID and prepare to
// receive update messages.
func synchronise(data []byte, c *Client) error {
	// Close previous update feed, if any
	if c.updateFeedCloser != nil && c.updateFeedCloser.IsOpen() {
		c.updateFeedCloser.Close()
		c.updateFeedCloser = nil
	}

	var msg syncMessage
	if err := decodeMessage(data, &msg); err != nil {
		return err
	}
	if !auth.IsBoard(msg.Board) {
		return errInvalidBoard
	}

	if msg.Thread == 0 {
		return syncToBoard(msg.Board, c)
	}

	return syncToThread(msg.Board, msg.Thread, msg.Ctr, c)
}

// Board pages do not have any live feeds (for now, at least). Just send the
// client its ID.
func syncToBoard(board string, c *Client) error {
	registerSync(board, c)
	return c.sendMessage(messageSynchronise, c.ID)
}

// Register the client with the central client storage datastructure
func registerSync(syncID string, c *Client) {
	if !c.synced {
		Clients.Add(c, syncID)
	} else {
		Clients.ChangeSync(c.ID, syncID)
	}
}

// Sends a response to the client's synchronisation request with any missed
// messages and starts streaming in updates.
func syncToThread(board string, thread, ctr int64, c *Client) error {
	valid, err := db.ValidateOP(thread, board)
	if err != nil {
		return err
	}
	if !valid {
		return errInvalidThread
	}

	closer := new(util.AtomicCloser)
	initial, err := db.StreamUpdates(thread, c.Send, closer)
	if err != nil {
		return err
	}

	// Guard against malicious counters, that result in out of bounds slicing
	// panic
	if int(ctr) < 0 || int(ctr) > len(initial) {
		closer.Close()
		return errInvalidCounter
	}

	c.updateFeedCloser = closer
	registerSync(util.IDToString(thread), c)

	// Send the client its ID
	if err := c.sendMessage(messageSynchronise, c.ID); err != nil {
		return err
	}

	// Send any messages the client is behind on
	for _, loggedMessage := range initial[ctr:] {
		if err := c.send(loggedMessage); err != nil {
			return err
		}
	}

	return nil
}

// Syncronise the client after a disconnect and restore any post in progress,
// if it is still not collected in the database
func resynchronise(data []byte, c *Client) error {

	// TODO: Open post restoration logic

	return synchronise(data, c)
}

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
