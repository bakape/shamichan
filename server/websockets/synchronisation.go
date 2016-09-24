// Syncronisation management message handlers

package websockets

import (
	"errors"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/db"
)

var (
	errInvalidBoard  = errors.New("invalid board")
	errInvalidThread = errors.New("invalid thread")
)

type syncRequest struct {
	Ctr    uint64 `json:"ctr"`
	Thread int64  `json:"thread"`
	Board  string `json:"board"`
}

// Syncronise the client to a certain thread, assign it's ID and prepare to
// receive update messages.
func synchronise(data []byte, c *Client) error {
	// Unsub from previous update feed, if any
	if err := c.closeFeed(); err != nil {
		return err
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

	return syncToThread(msg.Board, msg.Thread, msg.Ctr, c)
}

// Board pages do not have any live feeds (for now, at least). Just send the
// client its ID.
func syncToBoard(board string, c *Client) error {
	registerSync(board, 0, c)
	return c.sendMessage(messageSynchronise, 0)
}

// Register the client with the central client storage datastructure
func registerSync(board string, op int64, c *Client) {
	id := SyncID{
		OP:    op,
		Board: board,
	}
	if !c.synced {
		Clients.Add(c, id)
	} else {
		Clients.ChangeSync(c, id)
	}
}

// Sends a response to the client's synchronisation request with any missed
// messages and starts streaming in updates.
func syncToThread(board string, thread int64, ctr uint64, c *Client) error {
	valid, err := db.ValidateOP(thread, board)
	if err != nil {
		return err
	}
	if !valid {
		return errInvalidThread
	}

	registerSync(board, thread, c)

	var log [][]byte
	log, c.readFeed, c.cursor, err = streamUpdates(thread, ctr)
	if err != nil {
		return err
	}

	if err := c.sendMessage(messageSynchronise, 0); err != nil {
		return err
	}

	// Send all messages the client is behind on
	for _, msg := range log {
		if err := c.send(msg); err != nil {
			return err
		}
	}

	return nil
}

// Syncronise the client after a disconnect and restore any post in progress,
// if it is still not collected in the database
func resynchronise(data []byte, c *Client) error {

	// TODO: Open post restoration logic

	return synchronise(data, c)
}
