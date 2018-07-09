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
	auth.Captcha
}
type boardConfigSettingRequest struct {
	auth.Captcha
	config.BoardConfigs
}

type boardCreationRequest struct {
	auth.Captcha
	ID, Title string
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
	if !decodeJSON(w, r, &msg) {
		return
	}
	msg.ID = extractParam(r, "board")
	_, ok := canPerform(w, r, msg.ID, auth.BoardOwner, &msg.Captcha)
	if !ok || !validateBoardConfigs(w, msg.BoardConfigs) {
		return
	}
	if err := db.UpdateBoard(msg.BoardConfigs); err != nil {
		httpError(w, r, err)
		return
	}
}

// Assert user can perform a moderation action. If the action does not need a
// captcha verification, pass captcha as nil.
func canPerform(
	w http.ResponseWriter,
	r *http.Request,
	board string,
	level auth.ModerationLevel,
	captcha *auth.Captcha,
) (
	creds auth.SessionCreds, can bool,
) {
	if !auth.IsBoard(board) {
		httpError(w, r, errInvalidBoardName)
		return
	}
	ip, err := auth.GetIP(r)
	if err != nil {
		httpError(w, r, err)
		return
	}
	if captcha != nil && !auth.AuthenticateCaptcha(*captcha, ip, db.SystemBan) {
		httpError(w, r, errInvalidCaptcha)
		return
	}
	creds, ok := isLoggedIn(w, r)
	if !ok {
		return
	}

	can, err = db.CanPerform(creds.UserID, board, level)
	switch {
	case err != nil:
		httpError(w, r, err)
		return
	case !can:
		httpError(w, r, errAccessDenied)
		return
	default:
		can = true
		return
	}
}

// Assert client can moderate a post of unknown parenthood and return userID
func canModeratePost(
	w http.ResponseWriter,
	r *http.Request,
	id uint64,
	level auth.ModerationLevel,
) (
	board, userID string,
	can bool,
) {
	board, err := db.GetPostBoard(id)
	if err != nil {
		httpError(w, r, err)
		return
	}

	creds, can := canPerform(w, r, board, level, nil)
	if !can {
		httpError(w, r, errAccessDenied)
		return
	}

	userID = creds.UserID
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
	case len(conf.Notice) > common.MaxLenNotice:
		err = errNoticeTooLong
	case len(conf.Rules) > common.MaxLenRules:
		err = errRulesTooLong
	case len(conf.Title) > common.MaxLenBoardTitle:
		err = errTitleTooLong
	}
	if err == nil {
		matched := false
		for _, t := range common.Themes {
			if conf.DefaultCSS == t {
				matched = true
				break
			}
		}
		if !matched {
			err = errors.New("invalid default theme")
		}
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
		httpError(w, r, errAccessDenied)
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
		conf  config.BoardConfigs
		board = extractParam(r, "board")
	)
	if _, ok := canPerform(w, r, board, auth.BoardOwner, nil); !ok {
		return conf, false
	}

	conf = config.GetBoardConfigs(board).BoardConfigs
	conf.ID = board
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

	// Returns, if the board name, matches a reserved ID
	isReserved := func() bool {
		for _, s := range [...]string{"html", "json", "api", "assets", "all"} {
			if msg.ID == s {
				return true
			}
		}
		return false
	}

	// Validate request data
	ip, err := auth.GetIP(r)

	if err != nil {
		httpError(w, r, err)
		return
	}

	switch {
	case creds.UserID != "admin" && config.Get().DisableUserBoards:
		err = errAccessDenied
	case !boardNameValidation.MatchString(msg.ID),
		msg.ID == "",
		len(msg.ID) > common.MaxLenBoardID,
		isReserved():
		err = errInvalidBoardName
	case len(msg.Title) > 100:
		err = errTitleTooLong
	case !auth.AuthenticateCaptcha(msg.Captcha, ip, db.SystemBan):
		err = errInvalidCaptcha
	}
	if err != nil {
		httpError(w, r, err)
		return
	}

	err = db.InTransaction(func(tx *sql.Tx) (err error) {
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
			httpError(w, r, err)
			return
		default:
			httpError(w, r, err)
			return
		}

		return db.WriteStaff(tx, msg.ID, map[string][]string{
			"owners": []string{creds.UserID},
		})
	})

	switch err {
	case errBoardNameTaken, nil:
	default:
		httpError(w, r, err)
		return
	}

	err = db.WritePyu(msg.ID)
	if err != nil {
		httpError(w, r, err)
		return
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
		httpError(w, r, err)
	}
}

// Delete a board owned by the client
func deleteBoard(w http.ResponseWriter, r *http.Request) {
	var msg boardActionRequest
	if !decodeJSON(w, r, &msg) {
		return
	}
	creds, ok := canPerform(w, r, msg.Board, auth.BoardOwner, &msg.Captcha)
	if !ok {
		return
	}

	if err := db.DeleteBoard(msg.Board, creds.UserID); err != nil {
		httpError(w, r, err)
	}
}

// Delete one or multiple posts on a moderated board
func deletePost(w http.ResponseWriter, r *http.Request) {
	moderatePosts(w, r, auth.Janitor, db.DeletePost)
}

// Perform a moderation action an a single post. If ok == false, the caller
// should return.
func moderatePost(
	w http.ResponseWriter,
	r *http.Request,
	id uint64,
	level auth.ModerationLevel,
	fn func(userID string) error,
) (
	ok bool,
) {
	_, userID, can := canModeratePost(w, r, id, level)
	if !can {
		return
	}

	err := fn(userID)
	if err != nil {
		httpError(w, r, err)
	}
	return err == nil
}

// Same as moderatePost, but works on an array of posts
func moderatePosts(
	w http.ResponseWriter,
	r *http.Request,
	level auth.ModerationLevel,
	fn func(id uint64, userID string) error,
) {
	var ids []uint64
	if !decodeJSON(w, r, &ids) {
		return
	}
	for _, id := range ids {
		ok := moderatePost(w, r, id, auth.Janitor, func(userID string) error {
			return fn(id, userID)
		})
		if !ok {
			return
		}
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

// Ban a specific IP from a specific board
func ban(w http.ResponseWriter, r *http.Request) {
	var msg struct {
		Global   bool
		Duration uint64
		Reason   string
		IDs      []uint64
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
		httpError(w, r, errAccessDenied)
		return
	case len(msg.Reason) > common.MaxLenReason:
		httpError(w, r, errReasonTooLong)
		return
	case msg.Reason == "":
		httpError(w, r, errNoReason)
		return
	case msg.Duration == 0:
		httpError(w, r, errNoDuration)
		return
	}

	// Group posts by board
	byBoard := make(map[string][]uint64, 2)
	if msg.Global {
		byBoard["all"] = msg.IDs
	} else {
		for _, id := range msg.IDs {
			board, err := db.GetPostBoard(id)
			if err != nil {
				httpError(w, r, err)
				return
			}
			byBoard[board] = append(byBoard[board], id)
		}

		// Assert rights to moderate for all affected boards
		for b := range byBoard {
			if _, ok := canPerform(w, r, b, auth.Moderator, nil); !ok {
				return
			}
		}
	}

	// Apply bans
	expires := time.Now().Add(time.Duration(msg.Duration) * time.Minute)
	for board, ids := range byBoard {
		err := db.Ban(board, msg.Reason, creds.UserID, expires, true, ids...)
		if err != nil {
			httpError(w, r, err)
			return
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
		httpError(w, r, err)
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
	if !decodeJSON(w, r, &msg) {
		return
	}
	_, ok := canPerform(w, r, msg.Board, auth.BoardOwner, &msg.Captcha)
	if !ok {
		return
	}
	switch {
	// Ensure there always is at least one board owner
	case len(msg.Owners) == 0:
		httpError(w, r, common.ErrInvalidInput("no board owners set"))
		return
	default:
		// Maximum of 100 staff per position
		for _, s := range [...][]string{msg.Owners, msg.Moderators, msg.Janitors} {
			if len(s) > 100 {
				httpError(w, r, common.ErrInvalidInput(
					"too many staff per position"))
				return
			}
		}
	}

	err := db.InTransaction(func(tx *sql.Tx) error {
		return db.WriteStaff(tx, msg.Board, map[string][]string{
			"owners":     msg.Owners,
			"moderators": msg.Moderators,
			"janitors":   msg.Janitors,
		})
	})
	if err != nil {
		httpError(w, r, err)
		return
	}
}

// Retrieve posts with the same IP on the target board
func getSameIPPosts(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(extractParam(r, "id"), 10, 64)

	if err != nil {
		httpError(w, r, err)
		return
	}

	board, _, ok := canModeratePost(w, r, id, auth.Janitor)

	if !ok {
		return
	}

	creds, ok := isLoggedIn(w, r)

	if !ok {
		return
	}

	posts, err := db.GetSameIPPosts(id, board, creds)

	if err != nil {
		httpError(w, r, err)
		return
	}

	serveJSON(w, r, "", posts)
}

// Set the sticky flag of a thread
func setThreadSticky(w http.ResponseWriter, r *http.Request) {
	handleBoolRequest(w, r, func(id uint64, val bool, _ string) error {
		return db.SetThreadSticky(id, val)
	})
}

// Handle moderation request, that takes a boolean parameter,
// fn is the database call to be used for performing this operation.
func handleBoolRequest(
	w http.ResponseWriter,
	r *http.Request,
	fn func(id uint64, val bool, userID string) error,
) {
	var msg struct {
		ID  uint64
		Val bool
	}
	if !decodeJSON(w, r, &msg) {
		return
	}

	_, userID, ok := canModeratePost(w, r, msg.ID, auth.Moderator)
	if !ok {
		return
	}

	err := fn(msg.ID, msg.Val, userID)
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

	canUnban := detectCanPerform(r, board, auth.Moderator)
	html := []byte(templates.BanList(bans, board, canUnban))
	serveHTML(w, r, "", html, nil)
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
	board := extractParam(r, "board")
	creds, ok := canPerform(w, r, board, auth.Moderator, nil)
	if !ok {
		return
	}

	// Extract post IDs from form
	r.Body = http.MaxBytesReader(w, r.Body, jsonLimit)
	err := r.ParseForm()
	if err != nil {
		httpError(w, r, err)
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
			httpError(w, r, err)
			return
		}
		ids = append(ids, id)
	}

	// Unban posts
	for _, id := range ids {
		switch err := db.Unban(board, id, creds.UserID); err {
		case nil, sql.ErrNoRows:
		default:
			httpError(w, r, err)
			return
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/%s/", board), 303)
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

	serveHTML(w, r, "", []byte(templates.ModLog(log)), nil)
}

// Decodes params for client forced redirection. If ok = false , caller
// should abort.
func decodeRedirect(w http.ResponseWriter, r *http.Request) (
	id uint64, address string, ok bool,
) {
	var msg struct {
		ID  uint64
		URL string
	}
	if _, can := canPerform(w, r, "all", auth.Admin, nil); !can {
		return
	}
	if !decodeJSON(w, r, &msg) {
		return
	}
	return msg.ID, msg.URL, true
}

// Redirect all clients with the same IP as the target post to a URL
func redirectByIP(w http.ResponseWriter, r *http.Request) {
	id, url, ok := decodeRedirect(w, r)
	if !ok {
		return
	}

	ip, err := db.GetIP(id)
	if err == sql.ErrNoRows || ip == "" {
		text404(w)
		return
	} else if err != nil {
		httpError(w, r, err)
		return
	}

	msg, err := common.EncodeMessage(common.MessageRedirect, url)
	if err != nil {
		httpError(w, r, err)
		return
	}
	for _, c := range feeds.GetByIP(ip) {
		c.Send(msg)
	}
}

// Redirect all clients in the same thread to a URL
func redirectByThread(w http.ResponseWriter, r *http.Request) {
	id, url, ok := decodeRedirect(w, r)
	if !ok {
		return
	}

	msg, err := common.EncodeMessage(common.MessageRedirect, url)
	if err != nil {
		httpError(w, r, err)
		return
	}
	for _, c := range feeds.GetByThread(id) {
		c.Send(msg)
	}
}
