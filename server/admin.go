// Various administration endpoints for logged in users

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/dancannon/gorethink"
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
	errEightballTooLong = common.ErrTooLong("eightball")
	errTitleTooLong     = common.ErrTooLong("board title")
	errNoticeTooLong    = common.ErrTooLong("notice")
	errRulesTooLong     = common.ErrTooLong("rules")
	errInvalidBoardName = errors.New("invalid board name")
	errBoardNameTaken   = errors.New("board name taken")
	errAccessDenied     = errors.New("access denied")

	boardNameValidation = regexp.MustCompile(`^[a-z0-9]{1,3}$`)
)

// Request to set the board-specific configuration for a board
type boardConfigSettingRequest struct {
	sessionCreds
	config.BoardConfigs
}

// Request for the current non-public board configuration
type boardConfigRequest struct {
	sessionCreds
	ID string `json:"id"`
}

type configSettingRequest struct {
	sessionCreds
	config.Configs
}

type boardCreationRequest struct {
	Name, Title string
	sessionCreds
	common.Captcha
}

type boardDeletionRequest struct {
	ID string
	sessionCreds
	common.Captcha
}

type postDeletionRequest struct {
	IDs   []uint64 `json:"ids"`
	Board string
	sessionCreds
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
func configureBoard(w http.ResponseWriter, r *http.Request) {
	var msg boardConfigSettingRequest
	isValid := decodeJSON(w, r, &msg) &&
		isLoggedIn(w, r, msg.UserID, msg.Session) &&
		isBoardOwner(w, r, msg.ID, msg.UserID) &&
		validateBoardConfigs(w, msg.BoardConfigs)
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

	q := gorethink.Table("boards").Get(msg.ID).Update(conf)
	if err := db.Write(q); err != nil {
		text500(w, r, err)
		return
	}
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
func validateBoardConfigs(
	w http.ResponseWriter,
	conf config.BoardConfigs,
) bool {
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
	conf, isValid := boardConfData(w, r)
	if !isValid {
		return
	}
	serveJSON(w, r, "", conf)
}

// Serve the current server configurations. Available only to the "admin"
// account
func servePrivateServerConfigs(w http.ResponseWriter, r *http.Request) {
	var msg sessionCreds
	if !decodeJSON(w, r, &msg) || !isAdmin(w, r, msg) {
		return
	}
	serveJSON(w, r, "", config.Get())
}

func isAdmin(
	w http.ResponseWriter,
	r *http.Request,
	msg sessionCreds,
) bool {
	if !(isLoggedIn(w, r, msg.UserID, msg.Session)) {
		return false
	}
	if msg.UserID != "admin" {
		text403(w, errAccessDenied)
		return false
	}
	return true
}

// Determine, if the client has access rights to the configurations, and return
// them, if so
func boardConfData(w http.ResponseWriter, r *http.Request) (
	config.BoardConfigs, bool,
) {
	var (
		msg  boardConfigRequest
		conf config.BoardConfigs
	)
	isValid := decodeJSON(w, r, &msg) &&
		isLoggedIn(w, r, msg.UserID, msg.Session) &&
		isBoardOwner(w, r, msg.ID, msg.UserID)
	if !isValid {
		return conf, false
	}

	conf = config.GetBoardConfigs(msg.ID).BoardConfigs
	if conf.ID == "" {
		text404(w)
		return conf, false
	}

	return conf, true
}

// Handle requests to create a board
func createBoard(w http.ResponseWriter, r *http.Request) {
	var msg boardCreationRequest
	valid := decodeJSON(w, r, &msg) &&
		isLoggedIn(w, r, msg.UserID, msg.Session)
	if !valid {
		return
	}

	// Validate request data
	var err error
	switch {
	// "id" is a reserved key name in the database
	case msg.Name == "id", !boardNameValidation.MatchString(msg.Name):
		err = errInvalidBoardName
	case len(msg.Title) > 100:
		err = errTitleTooLong
	case !auth.AuthenticateCaptcha(msg.Captcha, auth.GetIP(r)):
		err = errInvalidCaptcha
	}
	if err != nil {
		text400(w, err)
		return
	}

	q := gorethink.Table("boards").Insert(config.DatabaseBoardConfigs{
		Created: time.Now(),
		BoardConfigs: config.BoardConfigs{
			BoardPublic: config.BoardPublic{
				Title:   msg.Title,
				Spoiler: "default.jpg",
				Banners: []string{},
			},
			ID:        msg.Name,
			Eightball: config.EightballDefaults,
			Staff: map[string][]string{
				"owners": []string{msg.UserID},
			},
		},
	})

	err = db.Write(q)
	switch {
	case gorethink.IsConflictErr(err):
		text400(w, errBoardNameTaken)
	case err != nil:
		text500(w, r, err)
	}
}

// Set the server configuration to match the one sent from the admin account
// user
func configureServer(w http.ResponseWriter, r *http.Request) {
	var msg configSettingRequest
	if !decodeJSON(w, r, &msg) || !isAdmin(w, r, msg.sessionCreds) {
		return
	}

	q := db.GetMain("config").
		Replace(func(doc gorethink.Term) gorethink.Term {
			return gorethink.Expr(msg.Configs).Merge(map[string]string{
				"id": "config",
			})
		})
	if err := db.Write(q); err != nil {
		text500(w, r, err)
	}
}

// Delete a board owned by the client
func deleteBoard(w http.ResponseWriter, r *http.Request) {
	var msg boardDeletionRequest
	isValid := decodeJSON(w, r, &msg) &&
		isLoggedIn(w, r, msg.UserID, msg.Session) &&
		isBoardOwner(w, r, msg.ID, msg.UserID)
	if !isValid {
		return
	}
	if !auth.AuthenticateCaptcha(msg.Captcha, auth.GetIP(r)) {
		text403(w, errInvalidCaptcha)
		return
	}
	if err := db.DeleteBoard(msg.ID); err != nil {
		text500(w, r, err)
		return
	}
}

// Delete one or multiple posts on a moderated board
func deletePost(w http.ResponseWriter, r *http.Request) {
	var msg postDeletionRequest

	// TODO: More than board owners should be able to delete posts
	isValid := decodeJSON(w, r, &msg) &&
		isLoggedIn(w, r, msg.UserID, msg.Session) &&
		isBoardOwner(w, r, msg.Board, msg.UserID)
	if !isValid {
		return
	}

	// Cast IDs to interfaces
	ids := make([]interface{}, len(msg.IDs))
	for i := range ids {
		ids[i] = interface{}(msg.IDs[i])
	}

	q := gorethink.
		Table("posts").
		GetAll(ids...).
		Filter(gorethink.Row.Field("board").Eq(msg.Board)).
		Update(map[string]interface{}{
			"log": gorethink.Row.Field("log").Append(gorethink.
				Expr("12").
				Add(gorethink.Row.Field("id").CoerceTo("string")).
				CoerceTo("binary"),
			),
			"deleted":     true,
			"lastUpdated": gorethink.Now().ToEpochTime().Floor(),
		})
	if err := db.Write(q); err != nil {
		text500(w, r, err)
		return
	}
}
