// Syncronisation management message handlers

package websockets

import (
	"errors"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
)

var (
	errInvalidBoard   = errors.New("invalid board")
	errInvalidThread  = errors.New("invalid thread")
	errInvalidCounter = errors.New("invalid progress counter")
)

type syncRequest struct {
	Ctr    int64  `json:"ctr"`
	Thread int64  `json:"thread"`
	Board  string `json:"board"`
}

// Syncronise the client to a certain thread, assign it's ID and prepare to
// receive update messages.
func synchronise(data []byte, c *Client) error {
	// Close previous update feed, if any
	if c.closeUpdateFeed != nil {
		close(c.closeUpdateFeed)
		c.closeUpdateFeed = nil
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
	registerSync("b:"+board, c)
	return c.sendMessage(messageSynchronise, 0)
}

// Register the client with the central client storage datastructure
func registerSync(syncID string, c *Client) {
	if !c.synced {
		Clients.Add(c, syncID)
	} else {
		Clients.ChangeSync(c, syncID)
	}
}

// Sends a response to the client's synchronisation request with any missed
// messages and starts streaming in updates.
func syncToThread(board string, thread, ctr int64, c *Client) error {
	valid, err := db.ValidateOP(thread, board)
	if err != nil {
		return err
	}
	if !valid {
		return errInvalidThread
	}

	closeFeed := make(chan struct{})
	initial, err := db.StreamUpdates(thread, c.write, closeFeed)
	if err != nil {
		return err
	}

	// Guard against malicious counters, that result in out of bounds slicing
	// panic
	if int(ctr) < 0 || int(ctr) > len(initial) {
		close(closeFeed)
		return errInvalidCounter
	}

	c.closeUpdateFeed = closeFeed
	registerSync(util.IDToString(thread), c)

	if err := c.sendMessage(messageSynchronise, 0); err != nil {
		return err
	}

	// Send any messages the client is behind on
	for _, loggedMessage := range initial[ctr:] {
		if err := c.send(loggedMessage); err != nil {
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
