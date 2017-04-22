package websockets

import (
	"errors"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"meguca/parser"
	"meguca/websockets/feeds"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	errReadOnly          = errors.New("read only board")
	errInvalidImageToken = errors.New("invalid image token")
	errImageNameTooLong  = errors.New("image name too long")
	errNoTextOrImage     = errors.New("no text or image")
)

// ThreadCreationRequest contains data for creating a new thread
type ThreadCreationRequest struct {
	ReplyCreationRequest
	Subject, Board string
}

// ReplyCreationRequest contains common fields for both thread and reply
// creation
type ReplyCreationRequest struct {
	Image ImageRequest
	auth.SessionCreds
	auth.Captcha
	Name, Password, Body string
}

// ImageRequest contains data for allocating an image
type ImageRequest struct {
	Spoiler     bool
	Token, Name string
}

// Insert a new thread into the database
func (c *Client) insertThread(data []byte) (err error) {
	if err := c.closePreviousPost(); err != nil {
		return err
	}

	var req ThreadCreationRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}

	post, err := CreateThread(req, c.ip, true)
	switch err {
	case nil:
	case errInValidCaptcha:
		return c.sendMessage(common.MessagePostID, -1)
	default:
		return err
	}

	c.post.init(post.StandalonePost)
	return c.sendMessage(common.MessagePostID, post.ID)
}

// CreateThread creates a new tread and writes it to the database.
// open specifies, if the thread OP should stay open after creation.
func CreateThread(req ThreadCreationRequest, ip string, open bool) (
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

	post, err = constructPost(
		req.ReplyCreationRequest,
		conf.ForcedAnon,
		open,
		ip,
		req.Board,
	)
	if err != nil {
		return
	}
	subject, err := parser.ParseSubject(req.Subject)
	if err != nil {
		return
	}

	// Perform this last, so there are less dangling images because of any error
	hasImage := !conf.TextOnly && req.Image.Token != "" && req.Image.Name != ""
	if hasImage {
		img := req.Image
		post.Image, err = getImage(img.Token, img.Name, img.Spoiler)
		if err != nil {
			return
		}
	}

	id, err := db.NewPostID()
	if err != nil {
		return
	}
	post.ID = id
	post.OP = id

	err = db.InsertThread(subject, post)
	return
}

// CreatePost creates a new post and writes it to the database.
// open specifies, if the post should stay open after creation.
func CreatePost(
	op uint64,
	board, ip string,
	open bool,
	req ReplyCreationRequest,
) (
	post db.Post, msg []byte, err error,
) {
	switch {
	case auth.IsBanned(board, ip):
		err = errBanned
		return
	// For now, only noscript posts need a captcha
	case !open && !auth.AuthenticateCaptcha(req.Captcha):
		err = errInValidCaptcha
		return
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

	post, err = constructPost(
		req,
		conf.ForcedAnon,
		open,
		ip,
		board,
	)
	if err != nil {
		return
	}

	if hasImage {
		img := req.Image
		post.Image, err = getImage(img.Token, img.Name, img.Spoiler)
		if err != nil {
			return
		}
	}

	post.OP = op
	post.ID, err = db.NewPostID()
	if err != nil {
		return
	}

	msg, err = common.EncodeMessage(common.MessageInsertPost, post.Post)
	if err != nil {
		return
	}

	err = db.InsertPost(post)
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
	post, msg, err := CreatePost(op, board, c.ip, true, req)
	if err != nil {
		return
	}

	// Ensure the client knows the post ID, before the public post insertion
	// update message is sent
	err = c.sendMessage(common.MessagePostID, post.ID)
	if err != nil {
		return
	}

	c.post.init(post.StandalonePost)
	c.feed.InsertPost(post.StandalonePost, c.post.body, msg)

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
	forcedAnon, open bool,
	ip, board string,
) (
	post db.Post, err error,
) {
	post = db.Post{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				Time: time.Now().Unix(),
			},
			Board: board,
		},
		IP: ip,
	}

	if !forcedAnon {
		post.Name, post.Trip, err = parser.ParseName(req.Name)
		if err != nil {
			return
		}
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

	post.Links, post.Commands, err = parser.ParseBody(
		[]byte(req.Body),
		board,
	)
	if err != nil {
		return
	}
	post.Body = req.Body

	// Attach staff position title after validations
	if req.UserID != "" {
		post.Auth, err = db.FindPosition(board, req.UserID)
		if err != nil {
			return
		}

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

	if open {
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
	}

	return
}

// Performs some validations and retrieves processed image data by token ID.
// Embeds spoiler and image name in result struct. The last extension is
// stripped from the name.
func getImage(token, name string, spoiler bool) (img *common.Image, err error) {
	if len(name) > 200 {
		return nil, errImageNameTooLong
	}

	imgCommon, err := db.UseImageToken(token)
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
