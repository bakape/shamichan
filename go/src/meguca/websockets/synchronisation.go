// Synchronization management message handlers

package websockets

import (
	"errors"
	"meguca/auth"
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
	Thread uint64
	Board  string
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

	return c.registerSync(msg.Thread, msg.Board)
}

// Register fresh client sync or change from previous sync
func (c *Client) registerSync(id uint64, board string) (err error) {
	// Don't close OP's, as navigating to the thread is a natural part of
	// thread creation
	if c.post.id != 0 && c.post.id != c.post.op {
		err = c.closePreviousPost()
		if err != nil {
			return
		}
	}

	c.feed, err = feeds.SyncClient(c, id, board)
	if err != nil {
		return
	}

	// Still sending something for consistency, but there is no actual syncing
	// to board pages
	if id == 0 {
		return c.sendMessage(common.MessageSynchronise, nil)
	}
	return
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
