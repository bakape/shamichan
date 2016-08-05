package websockets

import (
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

// Sent to the thread of the post being linked and appended to its replication
// log. Then propagated to all listeners subscribed to this log.
type backlinkInsertionMessage struct {
	ID    int64  `json:"id"`
	OP    int64  `json:"op"`
	Board string `json:"board"`
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

	update := map[string]map[int64]types.Link{
		"backlinks": {
			id: {
				OP:    op,
				Board: board,
			},
		},
	}

	// 3rd level nesting update. Runs completely DB-side.
	q := r.
		Table("threads").
		GetAllByIndex("post", destID).
		Update(map[string]r.Term{
			"log":   r.Row.Field("log").Append(msg),
			"posts": r.Object(util.IDToString(destID), update),
		})

	return db.Write(q)
}
