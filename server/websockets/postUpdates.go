package websockets

import (
	"errors"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/parser"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

var (
	errNoPostOpen = errors.New("no post open")
)

// Shorthand. We use it a lot for update query construction.
type msi map[string]interface{}

// Sent to the thread of the post being linked and appended to its replication
// log. Then propagated to all listeners subscribed to this log.
type backlinkInsertionMessage struct {
	ID    int64  `json:"id"`
	OP    int64  `json:"op"`
	Board string `json:"board"`
}

// Append a rune to the body of the open post
func appendRune(data []byte, c *Client) error {
	if !c.hasPost() {
		return errNoPostOpen
	}
	if c.openPost.bodyLength+1 > parser.MaxLengthBody {
		return parser.ErrBodyTooLong
	}
	var char rune
	if err := decodeMessage(data, &char); err != nil {
		return err
	}

	if char == '\n' {
		return parseLine(c)
	}

	id := c.openPost.id
	msg, err := encodeMessage(messageAppend, [2]int64{id, int64(char)})
	if err != nil {
		return err
	}

	update := msi{
		"body": r.Row.
			Field("posts").
			Field(util.IDToString(id)).
			Field("body").
			Add(string(char)),
	}
	if err := c.updatePost(update, msg); err != nil {
		return err
	}
	c.openPost.WriteRune(char)
	c.openPost.bodyLength++

	return nil
}

// Helper for running post update queries on the current open post
func (c *Client) updatePost(update msi, msg []byte) error {
	q := r.
		Table("threads").
		Get(c.openPost.op).
		Update(createUpdate(c.openPost.id, update, msg))
	return db.Write(q)
}

// Helper for creating post update maps
func createUpdate(id int64, update msi, msg []byte) msi {
	return msi{
		"log": appendLog(msg),
		"posts": msi{
			util.IDToString(id): update,
		},
	}
}

// Shorthand for creating a replication log append query
func appendLog(msg []byte) r.Term {
	return r.Row.Field("log").Append(msg)
}

// TODO: Line parsing
func parseLine(c *Client) error {
	return nil
}

// Writes the location data of the post linking a post to the the post being
// linked
func writeBacklink(id, op int64, board string, destID int64) error {
	msg, err := encodeMessage(messageBacklink, backlinkInsertionMessage{
		ID:    id,
		OP:    op,
		Board: board,
	})
	if err != nil {
		return err
	}

	update := msi{
		"backlinks": msi{
			util.IDToString(id): types.Link{
				OP:    op,
				Board: board,
			},
		},
	}
	q := r.
		Table("threads").
		GetAllByIndex("post", destID).
		Update(createUpdate(destID, update, msg))

	return db.Write(q)
}
