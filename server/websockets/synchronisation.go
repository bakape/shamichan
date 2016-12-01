// Synchronization management message handlers

package websockets

import (
	"bytes"
	"errors"
	"strings"

	"unicode/utf8"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
	"golang.org/x/crypto/bcrypt"
)

var (
	errInvalidBoard   = errors.New("invalid board")
	errInvalidThread  = errors.New("invalid thread")
	errInvalidCounter = errors.New("invalid progress counter")
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
func synchronise(data []byte, c *Client) error {
	// Unsubscribe from previous update feed, if any
	if c.feedID != 0 {
		feeds.Remove <- subRequest{c.feedID, c}
		c.feedID = 0
	}

	var msg syncRequest
	if err := decodeMessage(data, &msg); err != nil {
		return err
	}
	if !auth.IsBoard(msg.Board) {
		return errInvalidBoard
	}

	if msg.Thread == 0 {
		return syncToBoard(msg.Board, c)
	}

	return syncToThread(msg.Board, msg.Thread, c)
}

// Board pages do not have any live feeds (for now, at least). Just send the
// client its ID.
func syncToBoard(board string, c *Client) error {
	registerSync(board, 0, c)
	return c.sendMessage(MessageSynchronise, map[string]string{})
}

// Register the client with the central client storage data structure
func registerSync(board string, op uint64, c *Client) {
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
func syncToThread(board string, thread uint64, c *Client) error {
	valid, err := db.ValidateOP(thread, board)
	if err != nil {
		return err
	}
	if !valid {
		return errInvalidThread
	}

	registerSync(board, thread, c)
	feeds.Add <- subRequest{thread, c}
	c.feedID = thread

	return nil
}

// Reclaim an open post after connection loss or navigating away.
//
// TODO: Technically there is no locking performed so a single post may be open
// by multiple clients. This opens us up to some exploits, but nothing severe.
// Still need to think of a solution.
func reclaimPost(data []byte, c *Client) error {
	if err := closePreviousPost(c); err != nil {
		return err
	}

	var req reclaimRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}

	var post common.DatabasePost
	err := db.One(db.FindPost(req.ID).Default(nil), &post)
	if err != nil && err != r.ErrEmptyResult {
		return err
	}
	if !post.Editing {
		return c.sendMessage(MessageReclaim, 1)
	}
	if err := auth.BcryptCompare(req.Password, post.Password); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return c.sendMessage(MessageReclaim, 1)
		}
		return err
	}

	iLast := strings.LastIndexByte(post.Body, '\n')
	if iLast == -1 {
		iLast = 0
	}
	c.openPost = openPost{
		hasImage:   post.Image != nil,
		Buffer:     *bytes.NewBufferString(post.Body[iLast:]),
		bodyLength: utf8.RuneCountInString(post.Body),
		id:         post.ID,
		op:         post.OP,
		time:       post.Time,
		board:      post.Board,
	}

	return c.sendMessage(MessageReclaim, 0)
}
