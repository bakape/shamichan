// Synchronization management message handlers

package websockets

import (
	"bytes"
	"errors"
	"meguca/auth"
	"meguca/common"
	"meguca/db"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

var (
	errInvalidBoard   = errors.New("invalid board")
	errInvalidThread  = errors.New("invalid thread")
	errInvalidCounter = errors.New("invalid progress counter")
	errBanned         = errors.New("you are banned from this board")
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
	// Unsubscribe from previous update feed, if any
	c.unsubscribeFeed()

	// Send current server time on first synchronization
	if !c.synced {
		err := c.sendMessage(common.MessageServerTime, time.Now().Unix())
		if err != nil {
			return err
		}
	}

	var msg syncRequest
	err := decodeMessage(data, &msg)
	switch {
	case err != nil:
		return err
	case !auth.IsBoard(msg.Board):
		return errInvalidBoard
	case auth.IsBanned(msg.Board, c.ip):
		return errBanned
	case msg.Thread == 0:
		return c.syncToBoard(msg.Board)
	default:
		return c.syncToThread(msg.Board, msg.Thread)
	}
}

// Unsubscribe from update feed, if any
func (c *Client) unsubscribeFeed() {
	if c.feed != nil {
		feeds.Remove(c.feed, c)
		c.feed = nil
	}
}

// Board pages do not have any live feeds (for now, at least). Just send the
// client its ID.
func (c *Client) syncToBoard(board string) error {
	c.registerSync(board, 0)
	return c.sendMessage(common.MessageSynchronise, nil)
}

// Register the client with the central client storage data structure
func (c *Client) registerSync(board string, op uint64) {
	id := SyncID{
		OP:    op,
		Board: board,
	}
	if !c.synced {
		Clients.add(c, id)
	} else {
		Clients.changeSync(c, id)
	}
}

// Sends a response to the client's synchronization request with any missed
// messages and starts streaming in updates.
func (c *Client) syncToThread(board string, thread uint64) (err error) {
	valid, err := db.ValidateOP(thread, board)
	switch {
	case err != nil:
		return
	case !valid:
		return errInvalidThread
	}

	c.registerSync(board, thread)
	c.feed, err = feeds.Add(thread, c)
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

	c.post = openPost{
		hasImage: post.Image != nil,
		len:      utf8.RuneCountInString(post.Body),
		id:       post.ID,
		op:       post.OP,
		time:     post.Time,
		board:    post.Board,
		bodyBuffer: bodyBuffer{
			Buffer: *bytes.NewBufferString(post.Body),
		},
	}
	c.feed.InsertPost(&c.post, nil)

	return c.sendMessage(common.MessageReclaim, 0)
}
