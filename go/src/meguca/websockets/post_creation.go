package websockets

import (
	"database/sql"
	"encoding/binary"
	"errors"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"meguca/geoip"
	"meguca/parser"
	"meguca/websockets/feeds"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bakape/mnemonics"
)

var (
	errReadOnly          = errors.New("read only board")
	errInvalidImageToken = errors.New("invalid image token")
	errImageNameTooLong  = errors.New("image name too long")
	errNoTextOrImage     = errors.New("no text or image")
)

// ThreadCreationRequest contains data for creating a new thread
type ThreadCreationRequest struct {
	NonLive bool
	ReplyCreationRequest
	Subject, Board string
}

// ReplyCreationRequest contains common fields for both thread and reply
// creation
type ReplyCreationRequest struct {
	Sage, Open bool
	Image      ImageRequest
	auth.SessionCreds
	auth.Captcha
	Name, Password, Body string
}

// ImageRequest contains data for allocating an image
type ImageRequest struct {
	Spoiler     bool
	Token, Name string
}

// CreateThread creates a new tread and writes it to the database.
// open specifies, if the thread OP should stay open after creation.
func CreateThread(req ThreadCreationRequest, ip string) (
	post db.Post, err error,
) {
	switch {
	case !auth.IsNonMetaBoard(req.Board):
		err = errInvalidBoard
		return
	case auth.IsBanned(req.Board, ip):
		err = errBanned
		return
	case !auth.AuthenticateCaptcha(req.Captcha):
		err = errInValidCaptcha
		return
	}

	conf, err := getBoardConfig(req.Board)
	if err != nil {
		return
	}

	post, err = constructPost(req.ReplyCreationRequest, conf, ip)
	if err != nil {
		return
	}
	subject, err := parser.ParseSubject(req.Subject)
	if err != nil {
		return
	}

	// Must ensure image token usage is done atomically, as not to cause
	// possible data races with unused image cleanup
	tx, err := db.StartTransaction()
	if err != nil {
		return
	}
	defer db.RollbackOnError(tx, &err)

	post.ID, err = db.NewPostID(tx)
	if err != nil {
		return
	}
	post.OP = post.ID
	if conf.PosterIDs {
		computePosterID(&post)
	}

	hasImage := !conf.TextOnly && req.Image.Token != "" && req.Image.Name != ""
	if hasImage {
		img := req.Image
		post.Image, err = getImage(tx, img.Token, img.Name, img.Spoiler)
		if err != nil {
			return
		}
	}

	err = db.InsertThread(tx, subject, conf.NonLive || req.NonLive, post)
	if err != nil {
		return
	}

	err = tx.Commit()
	return
}

// Compute thread-level poster identification mnemonic and assign to post
func computePosterID(p *db.Post) {
	salt := config.Get().Salt
	b := make([]byte, 0, len(salt)+len(p.IP)+8)
	b = append(b, salt...)
	b = append(b, p.IP...)
	binary.LittleEndian.PutUint64(b, p.OP)

	p.PosterID = mnemonic.FantasyName(b)
}

// CreatePost creates a new post and writes it to the database.
// open specifies, if the post should stay open after creation.
func CreatePost(
	op uint64,
	board, ip string,
	needCaptcha bool,
	req ReplyCreationRequest,
) (
	post db.Post, msg []byte, err error,
) {
	if auth.IsBanned(board, ip) {
		err = errBanned
		return

	}
	if needCaptcha {
		if !auth.AuthenticateCaptcha(req.Captcha) {
			err = errInValidCaptcha
			return
		} else if config.Get().Captcha {
			// Captcha solved - reset spam score.
			auth.ResetSpamScore(ip)
		}
	}

	conf, err := getBoardConfig(board)
	if err != nil {
		return
	}

	// Post must have either at least one character or an image to be allocated
	hasImage := !conf.TextOnly && req.Image.Token != "" && req.Image.Name != ""
	if req.Body == "" && !hasImage {
		err = errNoTextOrImage
		return
	}

	// Assert thread is not locked
	locked, err := db.CheckThreadLocked(op)
	switch {
	case err != nil:
		return
	case locked:
		err = errors.New("thread is locked")
		return
	}

	// Disable live updates, if thread is non-live
	if req.Open {
		var disabled bool
		disabled, err = db.CheckThreadNonLive(op)
		if err != nil {
			return
		}
		req.Open = !disabled
	}

	post, err = constructPost(req, conf, ip)
	if err != nil {
		return
	}

	post.OP = op
	if conf.PosterIDs {
		computePosterID(&post)
	}

	// Must ensure image token usage is done atomically, as not to cause
	// possible data races with unused image cleanup
	tx, err := db.StartTransaction()
	if err != nil {
		return
	}
	defer db.RollbackOnError(tx, &err)

	post.ID, err = db.NewPostID(tx)
	if err != nil {
		return
	}

	if hasImage {
		img := req.Image
		post.Image, err = getImage(tx, img.Token, img.Name, img.Spoiler)
		if err != nil {
			return
		}
	}

	msg, err = common.EncodeMessage(common.MessageInsertPost, post.Post)
	if err != nil {
		return
	}

	err = db.InsertPost(tx, post, req.Sage)
	if err != nil {
		return
	}

	err = tx.Commit()
	return
}

// Insert a new post into the database
func (c *Client) insertPost(data []byte) (err error) {
	err = c.closePreviousPost()
	if err != nil {
		return
	}

	var req ReplyCreationRequest
	err = decodeMessage(data, &req)
	if err != nil {
		return
	}

	_, op, board := feeds.GetSync(c)
	post, msg, err := CreatePost(op, board, c.ip, !auth.CanPost(c.ip), req)
	if err != nil {
		return
	}

	// Ensure the client knows the post ID, before the public post insertion
	// update message is sent
	err = c.sendMessage(common.MessagePostID, post.ID)
	if err != nil {
		return
	}

	if post.Editing {
		err = db.SetOpenBody(post.ID, []byte(post.Body))
		if err != nil {
			return
		}
		c.post.init(post.StandalonePost)
	}
	c.feed.InsertPost(post.StandalonePost, c.post.body, msg)

	score := auth.PostCreationScore + auth.CharScore*time.Duration(c.post.len)
	if post.Image != nil {
		score += auth.ImageScore
	}
	return c.incrementSpamScore(score)
}

// Reset the IP's spam score, by submitting a captcha
func (c *Client) submitCaptcha(data []byte) (err error) {
	var msg auth.Captcha
	err = decodeMessage(data, &msg)
	if err != nil {
		return
	}

	if !auth.AuthenticateCaptcha(msg) {
		return errInValidCaptcha
	}
	auth.ResetSpamScore(c.ip)
	return nil
}

// If the client has a previous post, close it silently
func (c *Client) closePreviousPost() error {
	if c.post.id != 0 {
		return c.closePost()
	}
	return nil
}

// Retrieve post-related board configurations
func getBoardConfig(board string) (conf config.BoardConfigs, err error) {
	conf = config.GetBoardConfigs(board).BoardConfigs
	if conf.ReadOnly {
		err = errReadOnly
	}
	return
}

// Construct the common parts of the new post for both threads and replies
func constructPost(
	req ReplyCreationRequest,
	conf config.BoardConfigs,
	ip string,
) (
	post db.Post, err error,
) {
	post = db.Post{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				Time: time.Now().Unix(),
				Sage: req.Sage,
				Body: req.Body,
			},
			Board: conf.ID,
		},
		IP: ip,
	}

	if !conf.ForcedAnon {
		post.Name, post.Trip, err = parser.ParseName(req.Name)
		if err != nil {
			return
		}
	}

	if conf.Flags {
		post.Flag = geoip.LookUp(ip)
	}

	if utf8.RuneCountInString(req.Body) > common.MaxLenBody {
		err = common.ErrBodyTooLong
		return
	}

	lines := 0
	for _, r := range req.Body {
		if r == '\n' {
			lines++
		}
	}
	if lines > common.MaxLinesBody {
		err = errTooManyLines
		return
	}

	// Attach staff position title after validations
	if req.UserID != "" {
		var pos auth.ModerationLevel
		pos, err = db.FindPosition(conf.ID, req.UserID)
		if err != nil {
			return
		}
		post.Auth = pos.String()

		var loggedIn bool
		loggedIn, err = db.IsLoggedIn(req.UserID, req.Session)
		if err != nil {
			return
		}
		if !loggedIn {
			err = common.ErrInvalidCreds
			return
		}
	}

	if req.Open {
		post.Editing = true

		// Posts that are committed in one action need not a password, as they
		// are closed on commit and can not be reclaimed
		err = parser.VerifyPostPassword(req.Password)
		if err != nil {
			return
		}
		post.Password, err = auth.BcryptHash(req.Password, 4)
		if err != nil {
			return
		}
	} else {
		post.Links, post.Commands, err = parser.ParseBody(
			[]byte(req.Body),
			conf.ID,
		)
		if err != nil {
			return
		}
	}

	return
}

// Performs some validations and retrieves processed image data by token ID.
// Embeds spoiler and image name in result struct. The last extension is
// stripped from the name.
func getImage(tx *sql.Tx, token, name string, spoiler bool) (
	img *common.Image, err error,
) {
	if len(name) > 200 {
		return nil, errImageNameTooLong
	}

	imgCommon, err := db.UseImageToken(tx, token)
	switch err {
	case nil:
	case db.ErrInvalidToken:
		return nil, errInvalidImageToken
	default:
		return nil, err
	}

	// Trim on the last dot in the file name, but also strip for .tar.gz and
	// .tar.xz as special cases.
	if i := strings.LastIndexByte(name, '.'); i != -1 {
		name = name[:i]
	}
	if strings.HasSuffix(name, ".tar") {
		name = name[:len(name)-4]
	}

	return &common.Image{
		ImageCommon: imgCommon,
		Spoiler:     spoiler,
		Name:        name,
	}, nil
}
