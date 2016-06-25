// Syncronisation management message handlers

package websockets

import (
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
)

var (
	errInvalidBoard   = errInvalidMessage("Invalid board")
	errInvalidThread  = errInvalidMessage("Invalid thread")
	errInvalidCounter = errInvalidMessage("Invalid progress counter")
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
	if c.updateFeedCloser != nil && c.updateFeedCloser.IsOpen() {
		c.updateFeedCloser.Close()
		c.updateFeedCloser = nil
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
	if err := registerSync(board, c); err != nil {
		return err
	}
	return c.sendMessage(messageSynchronise, c.ID)
}

// Register the client with the central client storage datastructure
func registerSync(syncID string, c *Client) error {
	if !c.synced {
		return Clients.Add(c, syncID)
	}
	Clients.ChangeSync(c.ID, syncID)
	return nil
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

	closer := new(util.AtomicCloser)
	initial, err := db.StreamUpdates(thread, c.Send, closer)
	if err != nil {
		return err
	}

	// Guard against malicious counters, that result in out of bounds slicing
	// panic
	if int(ctr) < 0 || int(ctr) > len(initial) {
		closer.Close()
		return errInvalidCounter
	}

	c.updateFeedCloser = closer
	registerSync(util.IDToString(thread), c)

	// Send the client its ID
	if err := c.sendMessage(messageSynchronise, c.ID); err != nil {
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
