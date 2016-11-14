// Various administration endpoints for logged in users

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/parser"
	r "github.com/dancannon/gorethink"
)

const (
	// Body size limit for POST request JSON. Should never exceed 32 KB.
	// Consider anything bigger an attack.
	jsonLimit = 1 << 15

	maxAnswers      = 100  // Maximum number of eightball answers
	maxEightballLen = 2000 // Total chars in eightball
	maxNoticeLen    = 500
	maxRulesLen     = 5000
	maxTitleLen     = 100
)

var (
	errTooManyAnswers   = errors.New("too many eightball answers")
	errEightballTooLong = parser.ErrTooLong("eightball")
	errTitleTooLong     = parser.ErrTooLong("board title")
	errNoticeTooLong    = parser.ErrTooLong("notice")
	errRulesTooLong     = parser.ErrTooLong("rules")
)

// Embed in every request that needs authentication
type loginCredentials struct {
	UserID, Session string
}

// Request to set the board-specific configuration for a board
type boardConfigSettingRequest struct {
	loginCredentials
	config.BoardConfigs
}

// Request for the current non-public board configuration
type boardConfigRequest struct {
	loginCredentials
	ID string `json:"id"`
}

// Decode JSON sent in a request with a read limit of 8 KB. Returns if the
// decoding succeeded.
func decodeJSON(w http.ResponseWriter, r *http.Request, dest interface{}) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, jsonLimit))
	if err := decoder.Decode(dest); err != nil {
		http.Error(w, fmt.Sprintf("400 %s", err), 400)
		logError(r, err)
		return false
	}
	return true
}

// Set board-specific configurations to the user's owned board
func configureBoard(w http.ResponseWriter, req *http.Request) {
	var msg boardConfigSettingRequest
	isValid := decodeJSON(w, req, &msg) &&
		isLoggedIn(w, req, msg.UserID, msg.Session) &&
		isBoardOwner(w, req, msg.ID, msg.UserID) &&
		validateConfigs(w, msg.BoardConfigs)
	if !isValid {
		return
	}

	// TODO: Some kind of upload scheme for spoilers and banners
	conf := msg.BoardConfigs
	conf.Spoiler = "default.jpg"
	conf.Banners = []string{}

	// TODO: Staff configuration
	conf.Staff = map[string][]string{
		"owners": {msg.UserID},
	}

	q := r.Table("boards").Get(msg.ID).Update(conf)
	if err := db.Write(q); err != nil {
		text500(w, req, err)
		return
	}
}

// Assert the user login session ID is valid
func isLoggedIn(
	w http.ResponseWriter,
	req *http.Request,
	user, session string,
) bool {
	var isValid bool
	q := r.
		Table("accounts").
		Get(user).
		Field("sessions").
		Map(func(session r.Term) r.Term {
			return session.Field("token")
		}).
		Contains(session).
		Default(false)
	if err := db.One(q, &isValid); err != nil {
		text500(w, req, err)
		return false
	}

	if !isValid {
		http.Error(w, "403 Invalid login credentials", 403)
		return false
	}

	return true
}

// Assert the user is one of the board's owners
func isBoardOwner(
	w http.ResponseWriter,
	req *http.Request,
	board, userID string,
) (isOwner bool) {
	if staff := config.GetBoardConfigs(board).Staff; staff != nil {
		for _, o := range staff["owners"] {
			if o == userID {
				isOwner = true
				break
			}
		}
	}

	if !isOwner {
		http.Error(w, "403 Not board owner", 403)
	}
	return
}

// Validate length limit compliance of various fields
func validateConfigs(w http.ResponseWriter, conf config.BoardConfigs) bool {
	totalLen := 0
	for _, answer := range conf.Eightball {
		totalLen += len(answer)
	}

	var err error
	switch {
	case len(conf.Eightball) > maxAnswers:
		err = errTooManyAnswers
	case totalLen > maxEightballLen:
		err = errEightballTooLong
	case len(conf.Notice) > maxNoticeLen:
		err = errNoticeTooLong
	case len(conf.Rules) > maxRulesLen:
		err = errRulesTooLong
	case len(conf.Title) > maxTitleLen:
		err = errTitleTooLong
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("400 %s", err), 400)
		return false
	}

	return true
}

// Serve the current board configurations to the client, including publically
// unexposed ones. Intended to be used before setting the the configs with
// configureBoard().
func servePrivateBoardConfigs(w http.ResponseWriter, r *http.Request) {
	var msg boardConfigRequest
	isValid := decodeJSON(w, r, &msg) &&
		isLoggedIn(w, r, msg.UserID, msg.Session) &&
		isBoardOwner(w, r, msg.ID, msg.UserID)
	if !isValid {
		return
	}

	conf := config.GetBoardConfigs(msg.ID)
	if conf.ID == "" {
		text404(w)
		return
	}
	serveJSON(w, r, "", conf.BoardConfigs)
}
