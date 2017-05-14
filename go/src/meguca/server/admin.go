// Various administration endpoints for logged in users

package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"meguca/lang"
	"meguca/templates"
	"meguca/websockets/feeds"
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
	errTooManyAnswers   = errors.New("too many eightball answers")
	errEightballTooLong = common.ErrTooLong("eightball")
	errTitleTooLong     = common.ErrTooLong("board title")
	errNoticeTooLong    = common.ErrTooLong("notice")
	errRulesTooLong     = common.ErrTooLong("rules")
	errBanReasonTooLong = common.ErrTooLong("ban reason")
	errInvalidBoardName = errors.New("invalid board name")
	errBoardNameTaken   = errors.New("board name taken")
	errAccessDenied     = errors.New("access denied")
	errNoReason         = errors.New("no reason provided")
	errNoDuration       = errors.New("no ban duration provided")

	boardNameValidation = regexp.MustCompile(`^[a-z0-9]{1,3}$`)
)

type boardActionRequest struct {
	Board string
	auth.Captcha
}

type boardConfigSettingRequest struct {
	boardActionRequest
	config.BoardConfigs
}

type boardCreationRequest struct {
	boardActionRequest
	Title string
}

// Request to perform a moderation action on a specific set of posts
type postActionRequest struct {
	IDs []uint64
	boardActionRequest
}

type singlePostActionRequest struct {
	ID uint64
	boardActionRequest
}

// Decode JSON sent in a request with a read limit of 8 KB. Returns if the
// decoding succeeded.
func decodeJSON(w http.ResponseWriter, r *http.Request, dest interface{}) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, jsonLimit))
	if err := decoder.Decode(dest); err != nil {
		http.Error(w, fmt.Sprintf("400 %s", err), 400)
		return false
	}
	return true
}

// Set board-specific configurations to the user's owned board
func configureBoard(w http.ResponseWriter, r *http.Request) {
	var msg boardConfigSettingRequest
	isValid := decodeJSON(w, r, &msg) &&
		canPerform(w, r, msg.boardActionRequest, db.BoardOwner, true) &&
		validateBoardConfigs(w, msg.BoardConfigs)
	if !isValid {
		return
	}

	msg.BoardConfigs.ID = msg.Board
	if err := db.UpdateBoard(msg.BoardConfigs); err != nil {
		text500(w, r, err)
		return
	}
}

// Assert user can perform a moderation action
func canPerform(
	w http.ResponseWriter,
	r *http.Request,
	msg boardActionRequest,
	level db.ModerationLevel,
	needCaptcha bool,
) (
	can bool,
) {
	if !auth.IsBoard(msg.Board) {
		text400(w, errInvalidBoardName)
		return
	}
	if needCaptcha && !auth.AuthenticateCaptcha(msg.Captcha) {
		text403(w, errInvalidCaptcha)
		return
	}
	creds, ok := isLoggedIn(w, r)
	if !ok {
		return
	}

	can, err := db.CanPerform(creds.UserID, msg.Board, level)
	switch {
	case err != nil:
		text500(w, r, err)
		return
	case !can:
		text403(w, errAccessDenied)
		return
	default:
		return true
	}
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
	case len(conf.Notice) > common.MaxLenNotice:
		err = errNoticeTooLong
	case len(conf.Rules) > common.MaxLenRules:
		err = errRulesTooLong
	case len(conf.Title) > common.MaxLenBoardTitle:
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
	conf, ok := boardConfData(w, r)
	if !ok {
		return
	}
	serveJSON(w, r, "", conf)
}

// Serve the current server configurations. Available only to the "admin"
// account
func servePrivateServerConfigs(w http.ResponseWriter, r *http.Request) {
	if isAdmin(w, r) {
		serveJSON(w, r, "", config.Get())
	}
}

func isAdmin(w http.ResponseWriter, r *http.Request) bool {
	creds, ok := isLoggedIn(w, r)
	if !ok {
		return false
	}
	if creds.UserID != "admin" {
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
		msg  boardActionRequest
		conf config.BoardConfigs
	)
	ok := decodeJSON(w, r, &msg) && canPerform(w, r, msg, db.BoardOwner, false)
	if !ok {
		return conf, false
	}

	conf = config.GetBoardConfigs(msg.Board).BoardConfigs
	conf.ID = msg.Board
	if conf.ID == "" {
		text404(w)
		return conf, false
	}

	return conf, true
}

// Handle requests to create a board
func createBoard(w http.ResponseWriter, r *http.Request) {
	var msg boardCreationRequest
	if !decodeJSON(w, r, &msg) {
		return
	}
	creds, ok := isLoggedIn(w, r)
	if !ok {
		return
	}

	// Validate request data
	var err error
	switch {
	case creds.UserID != "admin" && config.Get().DisableUserBoards:
		err = errAccessDenied
	case !boardNameValidation.MatchString(msg.Board):
		err = errInvalidBoardName
	case len(msg.Title) > 100:
		err = errTitleTooLong
	case !auth.AuthenticateCaptcha(msg.Captcha):
		err = errInvalidCaptcha
	}
	if err != nil {
		text400(w, err)
		return
	}

	tx, err := db.StartTransaction()
	if err != nil {
		text500(w, r, err)
		return
	}
	defer db.RollbackOnError(tx, &err)

	err = db.WriteBoard(tx, db.BoardConfigs{
		Created: time.Now(),
		BoardConfigs: config.BoardConfigs{
			BoardPublic: config.BoardPublic{
				Title: msg.Title,
			},
			ID:        msg.Board,
			Eightball: config.EightballDefaults,
		},
	})
	switch {
	case err == nil:
	case db.IsConflictError(err):
		text400(w, errBoardNameTaken)
		return
	default:
		text500(w, r, err)
		return
	}

	err = db.WriteStaff(tx, msg.Board, map[string][]string{
		"owners": []string{creds.UserID},
	})
	if err != nil {
		text500(w, r, err)
		return
	}
	if err := tx.Commit(); err != nil {
		text500(w, r, err)
	}
}

// Set the server configuration to match the one sent from the admin account
// user
func configureServer(w http.ResponseWriter, r *http.Request) {
	var msg config.Configs
	if !decodeJSON(w, r, &msg) || !isAdmin(w, r) {
		return
	}
	if err := db.WriteConfigs(msg); err != nil {
		text500(w, r, err)
	}
}

// Delete a board owned by the client
func deleteBoard(w http.ResponseWriter, r *http.Request) {
	var msg boardActionRequest
	isValid := decodeJSON(w, r, &msg) &&
		canPerform(w, r, msg, db.BoardOwner, true)
	if !isValid {
		return
	}

	if err := db.DeleteBoard(msg.Board); err != nil {
		text500(w, r, err)
	}
}

// Delete one or multiple posts on a moderated board
func deletePost(w http.ResponseWriter, r *http.Request) {
	var msg postActionRequest
	if !decodeJSON(w, r, &msg) {
		return
	}

	var err error
	for _, id := range msg.IDs {
		msg.Board, err = db.GetPostBoard(id)
		switch err {
		case nil:
		case sql.ErrNoRows:
			text400(w, err)
			return
		default:
			text500(w, r, err)
			return
		}

		if !canPerform(w, r, msg.boardActionRequest, db.Janitor, false) {
			return
		}

		err = db.DeletePost(msg.Board, id)
		switch err.(type) {
		case nil:
		case common.ErrInvalidPostID:
			text400(w, err)
			return
		default:
			text500(w, r, err)
			return
		}
	}
}

// Ban a specific IP from a specific board
func ban(w http.ResponseWriter, r *http.Request) {
	var msg struct {
		Global   bool
		Duration uint64
		Reason   string
		postActionRequest
	}

	// Decode and validate
	if !decodeJSON(w, r, &msg) {
		return
	}
	creds, ok := isLoggedIn(w, r)
	switch {
	case !ok:
		return
	case msg.Global && creds.UserID != "admin":
		text403(w, errAccessDenied)
		return
	case len(msg.Reason) > common.MaxBanReasonLength:
		text400(w, errBanReasonTooLong)
		return
	case msg.Reason == "":
		text400(w, errNoReason)
		return
	case msg.Duration == 0:
		text400(w, errNoDuration)
		return
	}

	// Group posts by board
	byBoard := make(map[string][]uint64, 2)
	if msg.Global {
		byBoard["all"] = msg.IDs
	} else {
		for _, id := range msg.IDs {
			board, err := db.GetPostBoard(id)
			switch err {
			case nil:
			case sql.ErrNoRows:
				text400(w, err)
				return
			default:
				text500(w, r, err)
				return
			}

			byBoard[board] = append(byBoard[board], id)
		}

		// Assert rights to moderate for all affected boards
		for b := range byBoard {
			msg.Board = b
			if !canPerform(w, r, msg.boardActionRequest, db.Moderator, false) {
				return
			}
		}
	}

	// Apply bans
	expires := time.Now().Add(time.Duration(msg.Duration) * time.Minute)
	for board, ids := range byBoard {
		ips, err := db.Ban(board, msg.Reason, creds.UserID, expires, ids...)
		if err != nil {
			text500(w, r, err)
			return
		}

		// Redirect all banned connected clients to the /all/ board
		for ip := range ips {
			for _, cl := range common.GetByIPAndBoard(ip, board) {
				cl.Redirect("all")
			}
		}
	}
}

// Send a textual message to all connected clients
func sendNotification(w http.ResponseWriter, r *http.Request) {
	var msg string
	if !decodeJSON(w, r, &msg) || !isAdmin(w, r) {
		return
	}

	data, err := common.EncodeMessage(common.MessageNotification, msg)
	if err != nil {
		text500(w, r, err)
		return
	}
	for _, cl := range feeds.All() {
		cl.Send(data)
	}
}

// Assign moderation staff to a board
func assignStaff(w http.ResponseWriter, r *http.Request) {
	var msg struct {
		boardActionRequest
		Owners, Moderators, Janitors []string
	}

	isValid := decodeJSON(w, r, &msg) &&
		canPerform(w, r, msg.boardActionRequest, db.BoardOwner, true)
	switch {
	case !isValid:
		return
	// Ensure there always is at least one board owner
	case len(msg.Owners) == 0:
		text400(w, errors.New("no board owners set"))
		return
	default:
		// Maximum of 100 staff per position
		for _, s := range [...][]string{msg.Owners, msg.Moderators, msg.Janitors} {
			if len(s) > 100 {
				text400(w, errors.New("too many staff per position"))
				return
			}
		}
	}

	// Write to database
	tx, err := db.StartTransaction()
	if err != nil {
		text500(w, r, err)
		return
	}
	defer db.RollbackOnError(tx, &err)

	err = db.WriteStaff(tx, msg.Board, map[string][]string{
		"owners":     msg.Owners,
		"moderators": msg.Moderators,
		"janitors":   msg.Janitors,
	})
	if err != nil {
		text500(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		text500(w, r, err)
	}
}

// Retrieve posts with the same IP on the target board
func getSameIPPosts(w http.ResponseWriter, r *http.Request) {
	var msg singlePostActionRequest
	isValid := decodeJSON(w, r, &msg) &&
		canPerform(w, r, msg.boardActionRequest, db.Moderator, false)
	if !isValid {
		return
	}

	posts, err := db.GetSameIPPosts(msg.ID, msg.Board)
	if err != nil {
		text500(w, r, err)
		return
	}
	serveJSON(w, r, "", posts)
}

// Set the sticky flag of a thread
func setThreadSticky(w http.ResponseWriter, r *http.Request) {
	var msg struct {
		Sticky bool
		singlePostActionRequest
	}
	isValid := decodeJSON(w, r, &msg) &&
		canPerform(w, r, msg.boardActionRequest, db.Moderator, false)
	if !isValid {
		return
	}

	switch err := db.SetThreadSticky(msg.ID, msg.Sticky); err {
	case nil:
	case sql.ErrNoRows:
		text400(w, err)
	default:
		text500(w, r, err)
	}
}

// Render list of bans on a board with unban links for authenticated staff
func banList(w http.ResponseWriter, r *http.Request, p map[string]string) {
	board := p["board"]
	if !auth.IsBoard(board) {
		text404(w)
		return
	}

	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}

	bans, err := db.GetBoardBans(board)
	if err != nil {
		text500(w, r, err)
		return
	}

	canUnban := detectCanPerform(r, board, db.Moderator)
	html := []byte(templates.BanList(bans, board, canUnban, lp.UI))
	serveHTML(w, r, "", html, nil)
}

// Detect, if a  client can perform moderation on a board. Unlike canPerform,
// this will not send any errors to the client, if no access rights detected.
func detectCanPerform(
	r *http.Request,
	board string,
	level db.ModerationLevel,
) (can bool) {
	creds := extractLoginCreds(r)
	if creds.UserID == "" || creds.Session == "" {
		return
	}

	ok, err := db.IsLoggedIn(creds.UserID, creds.Session)
	if err != nil {
		return
	}
	if !ok {
		return
	}

	can, err = db.CanPerform(creds.UserID, board, level)
	return
}

// Unban a specific board -> banned post combination
func unban(w http.ResponseWriter, r *http.Request, p map[string]string) {
	board := p["board"]
	msg := boardActionRequest{
		Board: board,
	}
	if !canPerform(w, r, msg, db.Moderator, false) {
		return
	}

	// Extract post IDs from form
	r.Body = http.MaxBytesReader(w, r.Body, jsonLimit)
	err := r.ParseForm()
	if err != nil {
		text400(w, err)
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
			text400(w, err)
			return
		}
		ids = append(ids, id)
	}

	// Unban posts
	for _, id := range ids {
		switch err := db.Unban(board, id); err {
		case nil, sql.ErrNoRows:
		default:
			text500(w, r, err)
			return
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/%s/", board), 303)
}
