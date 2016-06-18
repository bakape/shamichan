// Wecbsocket message handlers

package websockets

import (
	"encoding/json"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
)

// Error while parsing the message. Denotes that either the message does not
// follow the structural spec or contains optional fields in unsupported
// combinations.
type errInvalidMessage string

func (e errInvalidMessage) Error() string {
	return string(e)
}

var (
	errInvalidStructure = errInvalidMessage("Invalid message structure")
	errInvalidBoard     = errInvalidMessage("Invalid board")
	errInvalidThread    = errInvalidMessage("Invalid thread")
	errInvalidCounter   = errInvalidMessage("Invalid progress counter")
)

// Decode message JSON into the suplied type
func decodeMessage(data []byte, dest interface{}) error {
	err := json.Unmarshal(data, dest)
	if err != nil {
		return errInvalidStructure
	}
	return nil
}

type syncMessage struct {
	Ctr    int64  `json:"ctr"`
	Thread int64  `json:"thread"`
	Board  string `json:"board"`
}

// Syncronise the client to a certain thread, assign it's ID and prepare to
// receive update messages.
func (c *Client) synchronise(data []byte) error {
	// Close previous update feed, if any
	if c.updateFeedCloser != nil && c.updateFeedCloser.IsOpen() {
		c.updateFeedCloser.Close()
		c.updateFeedCloser = nil
	}

	var msg syncMessage
	if err := decodeMessage(data, &msg); err != nil {
		return err
	}
	if !auth.IsBoard(msg.Board) {
		return errInvalidBoard
	}

	if msg.Thread == 0 {
		return c.syncToBoard(msg.Board)
	}

	return c.syncToThread(msg.Board, msg.Thread, msg.Ctr)
}

// Board pages do not have any live feeds (for now, at least). Just send the
// client its ID.
func (c *Client) syncToBoard(board string) error {
	c.registerSync(board)
	return c.sendMessage(messageSynchronise, c.ID)
}

// Register the client with the central client storage datastructure
func (c *Client) registerSync(syncID string) {
	if !c.synced {
		Clients.Add(c, syncID)
	} else {
		Clients.ChangeSync(c.ID, syncID)
	}
}

// Sends a response to the client's synchronisation request with any missed
// messages and starts streaming in updates.
func (c *Client) syncToThread(board string, thread, ctr int64) error {
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
	c.registerSync(util.IDToString(thread))

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
func (c *Client) resynchronise(data []byte) error {

	// TODO: Open post restoration logic

	return c.synchronise(data)
}
