// Synchronization management message handlers

package websockets

import (
	"errors"
	"meguca/auth"
	"meguca/cache"
	"meguca/common"
	"meguca/db"
	"meguca/websockets/feeds"

	"golang.org/x/crypto/bcrypt"
)

var (
	errInvalidBoard  = errors.New("invalid board")
	errInvalidThread = errors.New("invalid thread")
	errBanned        = errors.New("you are banned from this board")
)

type syncRequest struct {
	NewProtocol, Last100, Catalog bool
	Page                          uint
	Thread                        uint64
	Board                         string
}

type reclaimRequest struct {
	ID       uint64
	Password string
}

// Synchronise the client to a certain thread, assign it's ID and prepare to
// receive update messages.
func (c *Client) synchronise(data []byte) error {
	var msg syncRequest
	err := decodeMessage(data, &msg)
	switch {
	case err != nil:
		return err
	case !auth.IsBoard(msg.Board):
		return errInvalidBoard
	case auth.IsBanned(msg.Board, c.ip):
		return errBanned
	case msg.Thread != 0:
		valid, err := db.ValidateOP(msg.Thread, msg.Board)
		switch {
		case err != nil:
			return err
		case !valid:
			return errInvalidThread
		}
	}

	c.mu.Lock()
	c.newProtocol = msg.NewProtocol
	c.last100 = msg.Last100
	c.mu.Unlock()
	return c.registerSync(msg)
}

// Register fresh client sync or change from previous sync
func (c *Client) registerSync(req syncRequest) (err error) {
	if c.post.id != 0 {
		err = c.closePreviousPost()
		if err != nil {
			return
		}
	}

	c.feed, err = feeds.SyncClient(c, req.Thread, req.Board)
	if err != nil || req.Thread != 0 {
		return
	}
	if !req.NewProtocol {
		return c.sendMessage(common.MessageSynchronise, nil)
	}

	// Send board post data over websocket
	key := cache.BoardKey(req.Board, int64(req.Page), !req.Catalog)
	var f cache.FrontEnd
	if req.Catalog {
		f = cache.CatalogFE
	} else {
		f = cache.BoardPageFE
	}
	json, _, _, err := cache.GetJSONAndData(key, f)
	return c.send(common.PrependMessageType(common.MessageSynchronise, json))
}

// Reclaim an open post after connection loss or navigating away.
//
// TODO: Technically there is no locking performed so a single post may be open
// by multiple clients. This opens us up to some exploits, but nothing severe.
// Still need to think of a solution.
func (c *Client) reclaimPost(data []byte) error {
	if err := c.closePreviousPost(); err != nil {
		return err
	}

	var req reclaimRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}

	hash, err := db.GetPostPassword(req.ID)
	switch {
	case err != nil:
		return err
	case hash == nil:
		return c.sendMessage(common.MessageReclaim, 1)
	}

	switch err = auth.BcryptCompare(req.Password, hash); err {
	case nil:
	case bcrypt.ErrMismatchedHashAndPassword:
		return c.sendMessage(common.MessageReclaim, 1)
	default:
		return err
	}

	post, err := db.GetPost(req.ID)
	switch {
	case err != nil:
		return err
	case !post.Editing:
		return c.sendMessage(common.MessageReclaim, 1)
	}

	c.post.init(post)
	c.feed.InsertPost(post, c.post.body, nil)

	return c.sendMessage(common.MessageReclaim, 0)
}
