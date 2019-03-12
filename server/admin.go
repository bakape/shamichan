// Various administration endpoints for logged in users

package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/templates"
	"github.com/bakape/meguca/websockets/feeds"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

const (
	// Body size limit for POST request JSON. Should never exceed 32 KB.
	// Consider anything bigger an attack.
	jsonLimit = 1 << 15

	maxAnswers      = 100  // Maximum number of eightball answers
	maxEightballLen = 2000 // Total chars in eightball
)

var (
	errEightballTooLong = common.ErrTooLong("eightball")
	errTitleTooLong     = common.ErrTooLong("board title")
	errNoticeTooLong    = common.ErrTooLong("notice")
	errRulesTooLong     = common.ErrTooLong("rules")
	errReasonTooLong    = common.ErrTooLong("reason")
	errTooManyAnswers   = common.ErrInvalidInput("too many eightball answers")
	errInvalidBoardName = common.ErrInvalidInput("invalid board name")
	errBoardNameTaken   = common.ErrInvalidInput("board name taken")
	errNoReason         = common.ErrInvalidInput("no reason provided")
	errNoDuration       = common.ErrInvalidInput("no ban duration provided")
	errAccessDenied     = common.ErrAccessDenied("missing permissions")

	boardNameValidation = regexp.MustCompile(`^[a-z0-9]{1,10}$`)
)

type boardActionRequest struct {
	Board string
}

type boardCreationRequest struct {
	auth.Captcha
	ID, Title string
}

// Decode JSON sent in a request with a read limit of 8 KB. Returns if the
// decoding succeeded.
func decodeJSON(w http.ResponseWriter, r *http.Request, dest interface{},
) (
	err error,
) {
	err = json.NewDecoder(io.LimitReader(r.Body, jsonLimit)).Decode(dest)
	if err != nil {
		err = common.StatusError{err, 400}
	}
	return
}

// Set board-specific configurations to the user's owned board
func configureBoard(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		var msg config.BoardConfigs
		err = decodeJSON(w, r, &msg)
		if err != nil {
			return
		}

		msg.ID = extractParam(r, "board")
		_, err = canPerform(w, r, msg.ID, auth.BoardOwner, true)
		if err != nil {
			return
		}

		err = validateBoardConfigs(w, msg)
		if err != nil {
			return
		}
		return db.UpdateBoard(msg)
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Assert user can perform a moderation action. If the action does not need a
// captcha verification, pass captcha as nil.
func canPerform(w http.ResponseWriter, r *http.Request, board string,
	level auth.ModerationLevel, needCaptcha bool,
) (
	creds auth.SessionCreds, err error,
) {
	if !auth.IsBoard(board) {
		err = errInvalidBoardName
		return
	}
	ip, err := auth.GetIP(r)
	if err != nil {
		return
	}
	if needCaptcha {
		var has bool
		has, err = db.SolvedCaptchaRecently(ip, time.Minute)
		if err != nil {
			return
		}
		if !has {
			err = errInvalidCaptcha
			return
		}
	}
	creds, err = isLoggedIn(w, r)
	if err != nil {
		return
	}

	can, err := db.CanPerform(creds.UserID, board, level)
	switch {
	case err != nil:
	case !can:
		err = errAccessDenied
	}
	return
}

// Assert client can moderate a post of unknown parenthood and return userID
func canModeratePost(w http.ResponseWriter, r *http.Request, id uint64,
	level auth.ModerationLevel,
) (
	board, userID string, err error,
) {
	board, err = db.GetPostBoard(id)
	if err != nil {
		return
	}

	creds, err := canPerform(w, r, board, level, false)
	if err != nil {
		return
	}
	userID = creds.UserID
	return
}

// Validate length limit compliance of various fields
func validateBoardConfigs(w http.ResponseWriter, conf config.BoardConfigs,
) (
	err error,
) {
	totalLen := 0
	for _, answer := range conf.Eightball {
		totalLen += len(answer)
	}
	switch {
	case totalLen > maxEightballLen:
		err = errEightballTooLong
	case len(conf.Eightball) > maxAnswers:
		err = errTooManyAnswers
	case len(conf.Notice) > common.MaxLenNotice:
		err = errNoticeTooLong
	case len(conf.Rules) > common.MaxLenRules:
		err = errRulesTooLong
	case len(conf.Title) > common.MaxLenBoardTitle:
		err = errTitleTooLong
	}
	if err != nil {
		return
	}

	matched := false
	for _, t := range common.Themes {
		if conf.DefaultCSS == t {
			matched = true
			break
		}
	}
	if !matched {
		err = common.ErrInvalidInput("invalid default theme")
	}
	return
}

// Serve the current board configurations to the client, including publically
// unexposed ones. Intended to be used before setting the the configs with
// configureBoard().
func servePrivateBoardConfigs(w http.ResponseWriter, r *http.Request) {
	conf, err := boardConfData(w, r)
	if err != nil {
		httpError(w, r, err)
		return
	}
	serveJSON(w, r, "", conf)
}

// Serve the current server configurations. Available only to the "admin"
// account
func servePrivateServerConfigs(w http.ResponseWriter, r *http.Request) {
	err := isAdmin(w, r)
	if err != nil {
		httpError(w, r, err)
		return
	}
	serveJSON(w, r, "", config.Get())
}

func isAdmin(w http.ResponseWriter, r *http.Request) (err error) {
	creds, err := isLoggedIn(w, r)
	if err != nil {
		return
	}
	if creds.UserID != "admin" {
		err = errAccessDenied
	}
	return
}

// Determine, if the client has access rights to the configurations, and return
// them, if so
func boardConfData(w http.ResponseWriter, r *http.Request,
) (
	conf config.BoardConfigs, err error,
) {
	board := extractParam(r, "board")
	_, err = canPerform(w, r, board, auth.BoardOwner, false)
	if err != nil {
		return
	}

	conf = config.GetBoardConfigs(board).BoardConfigs
	conf.ID = board
	if conf.ID == "" {
		err = errInvalidBoardName
	}
	return
}

// Handle requests to create a board
func createBoard(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		var msg boardCreationRequest
		err = decodeJSON(w, r, &msg)
		if err != nil {
			return
		}

		creds, err := isLoggedIn(w, r)
		if err != nil {
			return
		}

		// Validate request data
		switch {
		case creds.UserID != "admin" && config.Get().DisableUserBoards:
			err = errAccessDenied
		case !boardNameValidation.MatchString(msg.ID),
			msg.ID == "",
			len(msg.ID) > common.MaxLenBoardID,
			// Returns, if the board name, matches a reserved ID
			func() bool {
				for _, s := range [...]string{
					"html", "json", "api", "assets", "all",
				} {
					if msg.ID == s {
						return true
					}
				}
				return false
			}():
			err = errInvalidBoardName
		case len(msg.Title) > 100:
			err = errTitleTooLong
		}
		if err != nil {
			return
		}

		ip, err := auth.GetIP(r)
		if err != nil {
			return
		}
		has, err := db.SolvedCaptchaRecently(ip, time.Minute)
		if err != nil {
			return
		}
		if !has {
			err = errInvalidCaptcha
			return
		}

		err = db.InTransaction(false, func(tx *sql.Tx) (err error) {
			err = db.WriteBoard(tx, db.BoardConfigs{
				Created: time.Now().UTC(),
				BoardConfigs: config.BoardConfigs{
					BoardPublic: config.BoardPublic{
						Title:      msg.Title,
						DefaultCSS: config.Get().DefaultCSS,
					},
					ID:        msg.ID,
					Eightball: config.EightballDefaults,
				},
			})
			switch {
			case err == nil:
			case db.IsConflictError(err):
				err = errBoardNameTaken
				return
			default:
				return
			}

			return db.WriteStaff(tx, msg.ID, map[string][]string{
				"owners": []string{creds.UserID},
			})
		})
		if err != nil {
			return
		}

		err = db.WritePyu(msg.ID)
		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Set the server configuration to match the one sent from the admin account
// user
func configureServer(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		var msg config.Configs
		err = decodeJSON(w, r, &msg)
		if err != nil {
			return
		}
		err = isAdmin(w, r)
		if err != nil {
			return
		}

		if len(msg.CaptchaTags) < 3 {
			err = common.StatusError{errors.New("too few captcha tags"), 400}
			return
		}
		err = db.WriteConfigs(msg)
		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Delete a board owned by the client
func deleteBoard(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		var msg boardActionRequest
		err = decodeJSON(w, r, &msg)
		if err != nil {
			return
		}
		creds, err := canPerform(w, r, msg.Board, auth.BoardOwner, true)
		if err != nil {
			return
		}

		return db.DeleteBoard(msg.Board, creds.UserID)
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Delete one or multiple posts on a moderated board
func deletePost(w http.ResponseWriter, r *http.Request) {
	moderatePosts(w, r, auth.Janitor, db.DeletePost)
}

// Perform a moderation action an a single post. If ok == false, the caller
// should return.
func moderatePost(w http.ResponseWriter, r *http.Request, id uint64,
	level auth.ModerationLevel, fn func(userID string) error,
) (
	err error,
) {
	_, userID, err := canModeratePost(w, r, id, level)
	if err != nil {
		return
	}
	return fn(userID)
}

// Same as moderatePost, but works on an array of posts
func moderatePosts(w http.ResponseWriter, r *http.Request,
	level auth.ModerationLevel, fn func(id uint64, userID string) error,
) {
	err := func() (err error) {
		var ids []uint64
		err = decodeJSON(w, r, &ids)
		if err != nil {
			return
		}
		for _, id := range ids {
			err = moderatePost(w, r, id, auth.Janitor,
				func(userID string) error {
					return fn(id, userID)
				})
			if err != nil {
				return
			}
		}
		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Permanently delete an image from a post
func deleteImage(w http.ResponseWriter, r *http.Request) {
	moderatePosts(w, r, auth.Janitor, db.DeleteImage)
}

// Spoiler image as a moderator
func modSpoilerImage(w http.ResponseWriter, r *http.Request) {
	moderatePosts(w, r, auth.Janitor, db.ModSpoilerImage)
}

// Clear post contents and remove any uploaded image from the server
func purgePost(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		var msg struct {
			ID     uint64
			Reason string
		}
		err = decodeJSON(w, r, &msg)
		if err != nil {
			return
		}

		_, userID, err := canModeratePost(w, r, msg.ID, auth.Admin)
		if err != nil {
			return
		}
		return db.PurgePost(msg.ID, userID, msg.Reason)
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Ban a specific IP from a specific board
func ban(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		var msg struct {
			Global       bool
			ID, Duration uint64
			Reason       string
		}
		err = decodeJSON(w, r, &msg)
		if err != nil {
			return
		}

		creds, err := isLoggedIn(w, r)
		switch {
		case err != nil:
		case len(msg.Reason) > common.MaxLenReason:
			err = errReasonTooLong
		case msg.Reason == "":
			err = errNoReason
		case msg.Duration == 0:
			err = errNoDuration
		}
		if err != nil {
			return
		}

		var board string
		if msg.Global {
			board = "all"
		} else {
			board, err = db.GetPostBoard(msg.ID)
			if err != nil {
				return
			}
		}
		_, err = canPerform(w, r, board, auth.Moderator, false)
		if err != nil {
			return
		}

		// Apply ban
		return db.Ban(board, msg.Reason, creds.UserID,
			time.Minute*time.Duration(msg.Duration), msg.ID)
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Send a textual message to all connected clients
func sendNotification(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		var msg string
		err = decodeJSON(w, r, &msg)
		if err != nil {
			return
		}
		err = isAdmin(w, r)
		if err != nil {
			return
		}

		data, err := common.EncodeMessage(common.MessageNotification, msg)
		if err != nil {
			return
		}
		for _, cl := range feeds.All() {
			cl.Send(data)
		}
		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Assign moderation staff to a board
func assignStaff(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		var msg struct {
			boardActionRequest
			Owners, Moderators, Janitors []string
		}
		err = decodeJSON(w, r, &msg)
		if err != nil {
			return
		}
		_, err = canPerform(w, r, msg.Board, auth.BoardOwner, true)
		if err != nil {
			return
		}

		// Ensure there always is at least one board owner
		if len(msg.Owners) == 0 {
			return common.ErrInvalidInput("no board owners set")
		}
		// Maximum of 100 staff per position
		for _, s := range [...][]string{
			msg.Owners, msg.Moderators, msg.Janitors,
		} {
			if len(s) > 100 {
				return common.ErrInvalidInput("too many staff per position")
			}
		}

		return db.InTransaction(false, func(tx *sql.Tx) error {
			return db.WriteStaff(tx, msg.Board, map[string][]string{
				"owners":     msg.Owners,
				"moderators": msg.Moderators,
				"janitors":   msg.Janitors,
			})
		})
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Retrieve posts with the same IP on the target board
func getSameIPPosts(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		id, err := strconv.ParseUint(extractParam(r, "id"), 10, 64)
		if err != nil {
			err = common.StatusError{err, 400}
			return
		}

		board, uid, err := canModeratePost(w, r, id, auth.Janitor)
		if err != nil {
			return
		}

		posts, err := db.GetSameIPPosts(id, board, uid)
		if err != nil {
			return
		}
		serveJSON(w, r, "", posts)
		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Set the sticky flag of a thread
func setThreadSticky(w http.ResponseWriter, r *http.Request) {
	handleBoolRequest(w, r, func(id uint64, val bool, _ string) error {
		return db.SetThreadSticky(id, val)
	})
}

// Handle moderation request, that takes a boolean parameter,
// fn is the database call to be used for performing this operation.
func handleBoolRequest(w http.ResponseWriter, r *http.Request,
	fn func(id uint64, val bool, userID string) error,
) {
	err := func() (err error) {
		var msg struct {
			ID  uint64
			Val bool
		}
		err = decodeJSON(w, r, &msg)
		if err != nil {
			return
		}

		_, userID, err := canModeratePost(w, r, msg.ID, auth.Moderator)
		if err != nil {
			return
		}

		return fn(msg.ID, msg.Val, userID)
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Set the locked flag of a thread
func setThreadLock(w http.ResponseWriter, r *http.Request) {
	handleBoolRequest(w, r, db.SetThreadLock)
}

// Render list of bans on a board with unban links for authenticated staff
func banList(w http.ResponseWriter, r *http.Request) {
	board := extractParam(r, "board")
	if !auth.IsBoard(board) {
		text404(w)
		return
	}

	bans, err := db.GetBoardBans(board)
	if err != nil {
		httpError(w, r, err)
		return
	}

	setHTMLHeaders(w)
	templates.WriteBanList(w, bans, board,
		detectCanPerform(r, board, auth.Moderator))
}

// Detect, if a  client can perform moderation on a board. Unlike canPerform,
// this will not send any errors to the client, if no access rights detected.
func detectCanPerform(
	r *http.Request,
	board string,
	level auth.ModerationLevel,
) (
	can bool,
) {
	creds := extractLoginCreds(r)
	if creds.UserID == "" || creds.Session == "" {
		return
	}

	ok, err := db.IsLoggedIn(creds.UserID, creds.Session)
	if err != nil || !ok {
		return
	}

	can, err = db.CanPerform(creds.UserID, board, level)
	return
}

// Unban a specific board -> banned post combination
func unban(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		board := extractParam(r, "board")
		creds, err := canPerform(w, r, board, auth.Moderator, false)
		if err != nil {
			return
		}

		// Extract post IDs from form
		r.Body = http.MaxBytesReader(w, r.Body, jsonLimit)
		err = r.ParseForm()
		if err != nil {
			err = common.StatusError{err, 400}
			return
		}
		var (
			id  uint64
			ids = make([]uint64, 0, 32)
		)
		for key, vals := range r.Form {
			if len(vals) == 0 || vals[0] != "on" {
				continue
			}
			id, err = strconv.ParseUint(key, 10, 64)
			if err != nil {
				err = common.StatusError{err, 400}
				return
			}
			ids = append(ids, id)
		}

		// Unban posts
		for _, id := range ids {
			err = db.Unban(board, id, creds.UserID)
			switch err {
			case nil:
			case sql.ErrNoRows:
				err = nil
			default:
				return
			}
		}

		http.Redirect(w, r, fmt.Sprintf("/%s/", board), 303)
		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Serve moderation log for a specific board
func modLog(w http.ResponseWriter, r *http.Request) {
	board := extractParam(r, "board")
	if !auth.IsBoard(board) {
		text404(w)
		return
	}

	log, err := db.GetModLog(board)
	if err != nil {
		httpError(w, r, err)
		return
	}
	setHTMLHeaders(w)
	templates.WriteModLog(w, log)
}

// Decodes params for client forced redirection
func decodeRedirect(w http.ResponseWriter, r *http.Request) (
	id uint64, address string, err error,
) {
	var msg struct {
		ID  uint64
		URL string
	}
	err = decodeJSON(w, r, &msg)
	if err != nil {
		return
	}
	id = msg.ID
	address = msg.URL
	_, err = canPerform(w, r, "all", auth.Admin, false)
	return
}

// Redirect all clients with the same IP as the target post to a URL
func redirectByIP(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		id, url, err := decodeRedirect(w, r)
		if err != nil {
			return
		}

		ip, err := db.GetIP(id)
		if err != nil {
			if err == sql.ErrNoRows {
				err = common.StatusError{errors.New("no such post"), 404}
			}
			return
		}
		if ip == "" {
			return common.StatusError{errors.New("no IP on post"), 404}
		}

		msg, err := common.EncodeMessage(common.MessageRedirect, url)
		if err != nil {
			return
		}
		for _, c := range feeds.GetByIP(ip) {
			c.Send(msg)
		}
		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Redirect all clients in the same thread to a URL
func redirectByThread(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		id, url, err := decodeRedirect(w, r)
		if err != nil {
			return
		}

		msg, err := common.EncodeMessage(common.MessageRedirect, url)
		if err != nil {
			return
		}
		for _, c := range feeds.GetByThread(id) {
			c.Send(msg)
		}
		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}
