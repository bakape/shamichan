package websockets

import (
	"bytes"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/parser"
)

var (
	errReadOnly          = errors.New("read only board")
	errInvalidImageToken = errors.New("invalid image token")
	errNoImageName       = errors.New("no image name")
	errImageNameTooLong  = errors.New("image name too long")
	errNoTextOrImage     = errors.New("no text or image")
	errThreadIsLocked    = errors.New("thread is locked")
)

// Websocket message response codes
const (
	postCreated = iota
	invalidInsertionCaptcha
)

// ThreadCreationRequest contains data for creating a new thread
type ThreadCreationRequest struct {
	ReplyCreationRequest
	Subject, Board, Captcha string
}

// ReplyCreationRequest contains common fields for both thread and reply
// creation
type ReplyCreationRequest struct {
	Image ImageRequest
	auth.SessionCreds
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

	id, now, hasImage, err := ConstructThread(req, c.ip, false)
	switch err {
	case nil:
	case errInValidCaptcha:
		return c.sendMessage(common.MessagePostID, -1)
	default:
		return err
	}

	c.post = openPost{
		id:       id,
		op:       id,
		time:     now,
		board:    req.Board,
		hasImage: hasImage,
	}

	return c.sendMessage(common.MessagePostID, id)
}

// ConstructThread creates a new tread and writes it to the database. Returns
// the ID of the thread and its creation timestamp
func ConstructThread(req ThreadCreationRequest, ip string, parseBody bool) (
	id uint64, timeStamp int64, hasImage bool, err error,
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

	post, timeStamp, _, err := constructPost(
		req.ReplyCreationRequest,
		conf.ForcedAnon,
		parseBody,
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

	// Perform this last, so there are less dangling images because of an error
	hasImage = !conf.TextOnly && req.Image.Token != "" && req.Image.Name != ""
	if hasImage {
		img := req.Image
		post.Image, err = getImage(img.Token, img.Name, img.Spoiler)
		if err != nil {
			return
		}
	}

	id, err = db.NewPostID()
	if err != nil {
		return
	}
	post.ID = id
	post.OP = id

	err = db.InsertThread(subject, post)
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

	_, sync := Clients.GetSync(c)
	if auth.IsBanned(sync.Board, c.ip) {
		return errBanned
	}
	conf, err := getBoardConfig(sync.Board)
	if err != nil {
		return
	}

	// Post must have either at least one character or an image to be allocated
	hasImage := !conf.TextOnly && req.Image.Token != "" && req.Image.Name != ""
	if req.Body == "" && !hasImage {
		return errNoTextOrImage
	}

	locked, err := db.IsLocked(sync.OP)
	switch {
	case err != nil:
		return
	case locked:
		return errThreadIsLocked
	}

	post, now, bodyLength, err := constructPost(
		req,
		conf.ForcedAnon,
		true,
		c.ip,
		sync.Board,
	)
	if err != nil {
		return
	}

	post.OP = sync.OP
	post.ID, err = db.NewPostID()
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

	// Ensure the client knows the post ID before the public post insertion
	// update message is sent
	err = c.sendMessage(common.MessagePostID, post.ID)
	if err != nil {
		return
	}
	err = db.InsertPost(post)
	if err != nil {
		return
	}

	c.post = openPost{
		id:       post.ID,
		op:       sync.OP,
		time:     now,
		board:    sync.Board,
		Buffer:   *bytes.NewBufferString(post.Body),
		len:      bodyLength,
		hasImage: hasImage,
	}

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
	forcedAnon, parseBody bool,
	ip, board string,
) (
	post db.Post,
	now int64,
	bodyLength int,
	err error,
) {
	now = time.Now().Unix()
	post = db.Post{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				Editing: true,
				Time:    now,
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

	if parseBody {
		bodyLength = utf8.RuneCountInString(req.Body)
		if bodyLength > common.MaxLenBody {
			err = common.ErrBodyTooLong
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
	}

	err = parser.VerifyPostPassword(req.Password)
	if err != nil {
		return
	}

	post.Password, err = auth.BcryptHash(req.Password, 4)
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
