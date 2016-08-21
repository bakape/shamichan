package websockets

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager"
	"github.com/bakape/meguca/parser"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

var (
	errReadOnly          = errors.New("read only board")
	errInvalidImageToken = errors.New("invalid image token")
	errNoImageName       = errors.New("no image name")
	errImageNameTooLong  = errors.New("image name too long")
)

// Websocket message response codes
const (
	postCreated = iota
	invalidInsertionCaptcha
)

// Response to a thread creation request
type threadCreationResponse struct {
	Code int   `json:"code"`
	ID   int64 `json:"id"`
}

// Insert a new thread into the database
func insertThread(data []byte, c *Client) (err error) {
	var req types.ThreadCreationRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}
	if !auth.IsNonMetaBoard(req.Board) {
		return errInvalidBoard
	}
	if !authenticateCaptcha(req.Captcha, c.IP) {
		return c.sendMessage(messageInsertThread, threadCreationResponse{
			Code: invalidInsertionCaptcha,
		})
	}

	var conf config.PostParseConfigs
	if err := db.One(db.GetBoardConfig(req.Board), &conf); err != nil {
		return err
	}
	if conf.ReadOnly {
		return errReadOnly
	}

	now := time.Now().Unix()
	thread := types.DatabaseThread{
		BumpTime:  now,
		ReplyTime: now,
		Board:     req.Board,
	}
	post := types.DatabasePost{
		Post: types.Post{
			Editing: true,
			Time:    now,
		},
		IP: c.IP,
	}

	post.Name, post.Trip, err = parser.ParseName(req.Name)
	if err != nil {
		return err
	}
	thread.Subject, err = parser.ParseSubject(req.Subject)
	if err != nil {
		return err
	}
	if err := parser.VerifyPostPassword(req.Password); err != nil {
		return err
	}
	post.Password, err = auth.BcryptHash(req.Password, 6)
	if err != nil {
		return err
	}

	// Perform this last, so there are less dangling images because of an error
	if !conf.TextOnly {
		post.Image, err = getImage(req.ImageToken, req.ImageName, req.Spoiler)
		thread.ImageCtr = 1
		if err != nil {
			return err
		}
	}

	id, err := db.ReservePostID()
	if err != nil {
		return err
	}
	thread.ID = id
	post.ID = id
	thread.Posts = map[int64]types.DatabasePost{
		id: post,
	}

	if err := db.Write(r.Table("threads").Insert(thread)); err != nil {
		return err
	}
	if err := db.IncrementBoardCounter(req.Board); err != nil {
		return err
	}

	c.openPost = openPost{
		id:    id,
		op:    id,
		time:  now,
		board: req.Board,
	}

	msg := threadCreationResponse{
		Code: postCreated,
		ID:   id,
	}
	return c.sendMessage(messageInsertThread, msg)
}

// Syncronise to a newly-created thread
func syncToNewThread(id int64, c *Client) error {
	close(c.closeUpdateFeed)
	c.closeUpdateFeed = nil

	closeFeed := make(chan struct{})
	if _, err := db.StreamUpdates(id, c.write, closeFeed); err != nil {
		return err
	}

	c.closeUpdateFeed = closeFeed
	registerSync(util.IDToString(id), c)

	return c.sendMessage(messageSynchronise, 0)
}

// Performs some validations and retrieves processed image data by token ID.
// Embeds spoiler and image name in result struct. The last extension is
// stripped from the name.
func getImage(token, name string, spoiler bool) (img *types.Image, err error) {
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

	imgCommon, err := imager.UseImageToken(token)
	if err != nil {
		if err == imager.ErrInvalidToken {
			err = errInvalidImageToken
		}
		return
	}

	return &types.Image{
		ImageCommon: imgCommon,
		Spoiler:     spoiler,
		Name:        strings.TrimSuffix(name, filepath.Ext(name)),
	}, nil
}
