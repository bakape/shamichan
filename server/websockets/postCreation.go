package websockets

import (
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"

	"github.com/bakape/meguca/parser"
)

var (
	// Overridable for tests
	imageAllocationTimeout = time.Minute * 15

	errImageAllocationTimeout = errInvalidMessage("image allocation timeout")
	errReadOnly               = errInvalidMessage("read only board")
)

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
		return c.sendMessage(messageInsertThread, false)
	}

	var conf config.PostParseConfigs
	if err := db.One(db.GetBoardConfig(req.Board), &conf); err != nil {
		return err
	}
	if conf.ReadOnly {
		return errReadOnly
	}

	now := time.Now().Unix() * 1000
	thread := types.DatabaseThread{
		BumpTime:  now,
		ReplyTime: now,
		Board:     req.Board,
	}
	post := types.Post{
		Time:  now,
		Board: req.Board,
		IP:    c.IP,
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
	post.Password = req.Password

	bp := parser.BodyParser{
		Config: conf,
		Board:  req.Board,
	}
	res, err := bp.ParseBody(req.Body)
	if err != nil {
		return err
	}
	post.Body = res.Body
	post.Links = res.Links
	post.Commands = res.Commands

	if !conf.TextOnly {
		select {
		case img := <-c.AllocateImage:
			post.Image = &img
		case <-time.Tick(imageAllocationTimeout):
			return errImageAllocationTimeout
		}
	}

	id, err := db.ReservePostID()
	if err != nil {
		return err
	}
	thread.ID = id
	post.ID = id
	post.OP = id
	thread.Posts = map[int64]types.Post{
		id: post,
	}

	if err := db.Write(r.Table("threads").Insert(thread)); err != nil {
		return err
	}
	if err := db.IncrementBoardCounter(req.Board); err != nil {
		return err
	}
	if err := db.WriteBacklinks(id, id, req.Board, post.Links); err != nil {
		return err
	}

	return c.sendMessage(messageInsertThread, true)
}
