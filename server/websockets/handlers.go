// Wecbsocket message handlers

package websockets

import (
	"encoding/json"
	"github.com/bakape/meguca/db"
)

// Error while parsing the message. Denotes that either the message does not
// follow the structural spec or contains optional fields in unsupported
// combinations.
type errInvalidMessage string

func (e errInvalidMessage) Error() string {
	return string(e)
}

// Decode message JSON into the suplied type
func decodeMessage(data []byte, dest interface{}) error {
	err := json.Unmarshal(data, dest)
	if err != nil {
		return errInvalidMessage("Invalid message structure")
	}
	return nil
}

type syncMessage struct {
	Ctr    int64  `json:"ctr"`
	Thread int64  `json:"thread"`
	Board  string `json:"string"`
}

// Syncronise the client to a certain thread, assign it's ID and prepare to
// receive update messages.
func (c *Client) synchronise(data []byte) error {
	var msg syncMessage
	if err := decodeMessage(data, msg); err != nil {
		return err
	}
	valid, err := db.ValidateOP(msg.Thread, msg.Board)
	if err != nil {
		return err
	}
	if !valid {
		return errInvalidMessage("Invalid thread or board")
	}
	Clients.Add(c)
	// Subs.ListenTo(msg.Thread, c, msg.Ctr)
	return nil
}

// Syncronise the client after a disconnect and restore any post in progress,
// if it is still not collected in the database
func (c *Client) resynchronise(data []byte) error {

	// TODO: Open post restoration logic

	return c.synchronise(data)
}
