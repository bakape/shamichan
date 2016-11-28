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
	Image                      ImageRequest
	Name, Auth, Password, Body string
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

	id, now, hasImage, err := ConstructThread(req, c.ip, false)
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
		hasImage: hasImage,
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
	id, timeStamp int64, hasImage bool, err error,
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
	hasImage = !conf.TextOnly && req.Image.Token != "" && req.Image.Name != ""
	if hasImage {
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
	hasImage := !conf.TextOnly && req.Image.Token != "" && req.Image.Name != ""
	if req.Body == "" && !hasImage {
		return errNoTextOrImage
	}

	var locked bool
	q := r.Table("threads").Get(sync.OP).Field("locked").Default(false)
	if err := db.One(q, &locked); err != nil {
		return err
	}
	if locked {
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

	if hasImage {
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
		hasImage:   hasImage,
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
	case len(token) > 127: // RethinkDB key length limit
		return nil, errInvalidImageToken
	case len(name) > 200:
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

	// Trim on the first dot in the file name. Not using filepath.Ext(), because
	// it does not handle compound extensions like ".tar.gz"
	if i := strings.IndexByte(name, '.'); i != -1 {
		name = name[:i]
	}

	return &common.Image{
		ImageCommon: imgCommon,
		Spoiler:     spoiler,
		Name:        name,
	}, nil
}
