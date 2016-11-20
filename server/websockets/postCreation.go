package websockets

import (
	"bytes"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/parser"
	"github.com/bakape/meguca/common"
	r "github.com/dancannon/gorethink"
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
	Subject, Board string
	common.Captcha
}

// ReplyCreationRequest contains common fields for both thread and reply
// creation
type ReplyCreationRequest struct {
	Image                             ImageRequest
	Name, Email, Auth, Password, Body string
}

// ImageRequest contains data for allocating an image
type ImageRequest struct {
	Spoiler     bool
	Token, Name string
}

// Response to a thread creation request
type threadCreationResponse struct {
	Code int   `json:"code"`
	ID   int64 `json:"id"`
}

// Insert a new thread into the database
func insertThread(data []byte, c *Client) (err error) {
	if err := closePreviousPost(c); err != nil {
		return err
	}
	var req ThreadCreationRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}

	id, now, err := ConstructThread(req, c.IP, false)
	if err != nil {
		if err == errInValidCaptcha {
			return c.sendMessage(MessageInsertThread, threadCreationResponse{
				Code: invalidInsertionCaptcha,
			})
		}
		return err
	}

	c.openPost = openPost{
		id:       id,
		op:       id,
		time:     now,
		board:    req.Board,
		hasImage: req.Image.Token != "",
	}

	msg := threadCreationResponse{
		Code: postCreated,
		ID:   id,
	}
	return c.sendMessage(MessageInsertThread, msg)
}

// ConstructThread creates a new tread and writes it to the database. Returns
// the ID of the thread and its creation timestamp
func ConstructThread(req ThreadCreationRequest, ip string, parseBody bool) (
	id, timeStamp int64, err error,
) {
	if !auth.IsNonMetaBoard(req.Board) {
		err = errInvalidBoard
		return
	}
	if !auth.AuthenticateCaptcha(req.Captcha, ip) {
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
	thread := common.DatabaseThread{
		ReplyTime: timeStamp,
		Board:     req.Board,
	}
	thread.Subject, err = parser.ParseSubject(req.Subject)
	if err != nil {
		return
	}

	// Perform this last, so there are less dangling images because of an error
	if !conf.TextOnly {
		img := req.Image
		post.Image, err = getImage(img.Token, img.Name, img.Spoiler)
		thread.ImageCtr = 1
		if err != nil {
			return
		}
	}

	id, err = db.ReservePostID()
	if err != nil {
		return
	}
	thread.ID = id
	post.ID = id
	post.OP = id

	err = db.Insert("posts", post)
	if err != nil {
		return
	}
	err = db.Insert("threads", thread)
	if err != nil {
		return
	}
	err = db.IncrementBoardCounter(req.Board)
	if err != nil {
		return
	}

	return
}

// Insert a new post into the database
func insertPost(data []byte, c *Client) error {
	if err := closePreviousPost(c); err != nil {
		return err
	}

	var req ReplyCreationRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}

	_, sync := Clients.GetSync(c)
	conf, err := getBoardConfig(sync.Board)
	if err != nil {
		return err
	}

	// Post must have either at least one character or an image to be allocated
	noImage := conf.TextOnly || req.Image.Token == "" || req.Image.Name == ""
	if req.Body == "" && noImage {
		return errNoTextOrImage
	}

	// Check thread is not locked and retrieve the post counter
	var threadAttrs struct {
		Locked  bool
		PostCtr int
	}
	q := r.Table("threads").Get(sync.OP).Pluck("locked", "postCtr")
	if err := db.One(q, &threadAttrs); err != nil {
		return err
	}
	if threadAttrs.Locked {
		return errThreadIsLocked
	}

	post, now, bodyLength, err := constructPost(
		req,
		conf.ForcedAnon,
		true,
		c.IP,
		sync.Board,
	)
	if err != nil {
		return err
	}

	post.OP = sync.OP
	post.ID, err = db.ReservePostID()
	if err != nil {
		return err
	}

	updates := make(map[string]interface{}, 3)
	updates["postCtr"] = r.Row.Field("postCtr").Add(1)
	updates["replyTime"] = now

	if !noImage {
		img := req.Image
		post.Image, err = getImage(img.Token, img.Name, img.Spoiler)
		if err != nil {
			return err
		}
		updates["imageCtr"] = r.Row.Field("imageCtr").Add(1)
	}

	// Ensure the client knows the post ID before the public post insertion
	// update message is sent
	if err := c.sendMessage(MessagePostID, post.ID); err != nil {
		return err
	}

	if err := db.Insert("posts", post); err != nil {
		return err
	}
	q = r.Table("threads").Get(sync.OP).Update(updates)
	if err := db.Write(q); err != nil {
		return err
	}
	if err := db.IncrementBoardCounter(sync.Board); err != nil {
		return err
	}

	c.openPost = openPost{
		id:         post.ID,
		op:         sync.OP,
		time:       now,
		board:      sync.Board,
		Buffer:     *bytes.NewBufferString(lastLine(post.Body)),
		bodyLength: bodyLength,
		hasImage:   !noImage,
	}

	return nil
}

func lastLine(s string) string {
	i := strings.LastIndexByte(s, '\n')
	if i == -1 {
		return s
	}
	return s[i+1:]
}

// If the client has a previous post, close it silently
func closePreviousPost(c *Client) error {
	if c.openPost.id != 0 {
		return closePost(nil, c)
	}
	return nil
}

// Retrieve post-related board configurations
func getBoardConfig(board string) (conf config.PostParseConfigs, err error) {
	conf = config.GetBoardConfigs(board).PostParseConfigs
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
	post common.DatabasePost, now int64, bodyLength int, err error,
) {
	now = time.Now().Unix()
	post = common.DatabasePost{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				Editing: true,
				Time:    now,
				Email:   parser.FormatEmail(req.Email),
			},
			Board: board,
		},
		LastUpdated: now,
		IP:          ip,
	}

	if !forcedAnon {
		post.Name, post.Trip, err = parser.ParseName(req.Name)
		if err != nil {
			return
		}
	}

	if parseBody {
		buf := []byte(req.Body)
		bodyLength = utf8.RuneCount(buf)
		if bodyLength > common.MaxLenBody {
			err = common.ErrBodyTooLong
			return
		}
		post.Links, post.Commands, err = parser.ParseBody(buf, board)
		if err != nil {
			return
		}
		post.Body = req.Body
	}

	err = parser.VerifyPostPassword(req.Password)
	if err != nil {
		return
	}

	post.Password, err = auth.BcryptHash(req.Password, 6)

	// TODO: Staff title verification

	return
}

// Performs some validations and retrieves processed image data by token ID.
// Embeds spoiler and image name in result struct. The last extension is
// stripped from the name.
func getImage(token, name string, spoiler bool) (img *common.Image, err error) {
	switch {
	case len(token) > 127, token == "": // RethinkDB key length limit
		err = errInvalidImageToken
	case name == "":
		err = errNoImageName
	case len(name) > 200:
		err = errImageNameTooLong
	}
	if err != nil {
		return
	}

	imgCommon, err := db.UseImageToken(token)
	if err != nil {
		if err == db.ErrInvalidToken {
			err = errInvalidImageToken
		}
		return
	}

	// Trim on the first dot in the file name. Not using filepath.Ext(), because
	// it does not handle compound extensions like ".tar.gz"
	switch i := strings.IndexByte(name, '.'); i {
	case -1:
	default:
		name = name[:i]
	}

	return &common.Image{
		ImageCommon: imgCommon,
		Spoiler:     spoiler,
		Name:        name,
	}, nil
}
