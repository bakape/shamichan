package db

import (
	"database/sql"
	"encoding/json"

	"github.com/bakape/meguca/common"
)

// Message sent to all clients to inject a command result into a model
type commandMessage struct {
	ID uint64 `json:"id"`
	common.Command
}

// Message sent to listening clients about a link or backlink insertion into
// a post
type linkMessage struct {
	ID    uint64      `json:"id"`
	Links [][2]uint64 `json:"links"`
}

// Message that signals and insertion of an image into an existing post
type imageMessage struct {
	common.Image
	ID uint64 `json:"id"`
}

// BumpThread dumps up thread counters and adds a message to the thread's
// replication log
func BumpThread(
	tx *sql.Tx,
	id uint64,
	reply, bump, image bool,
	msg []byte,
) error {
	_, err := tx.Stmt(prepared["bumpThread"]).Exec(id, reply, bump, image)
	if err != nil {
		return err
	}
	return UpdateLog(tx, id, msg)
}

// UpdateLog writes to a thread's replication log
func UpdateLog(tx *sql.Tx, id uint64, msg []byte) error {
	_, err := tx.Stmt(prepared["updateLog"]).Exec(id, msg)
	return err
}

// AppendBody appends a character to a post body
func AppendBody(id, op uint64, char rune) error {
	msg, err := common.EncodeMessage(
		common.MessageAppend,
		[2]uint64{id, uint64(char)},
	)
	if err != nil {
		return err
	}
	return updatePost(id, op, msg, "appendBody", string(char))
}

func updatePost(
	id, op uint64,
	msg []byte,
	queryKey string,
	arg interface{},
) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer RollbackOnError(tx, &err)

	q := tx.Stmt(prepared[queryKey])
	if arg != nil {
		_, err = q.Exec(id, arg)
	} else {
		_, err = q.Exec(id)
	}
	if err != nil {
		return
	}
	err = UpdateLog(tx, op, msg)
	if err != nil {
		return
	}

	return tx.Commit()
}

// InsertCommand inserts a has command result into a post
func InsertCommand(id, op uint64, com common.Command) error {
	msg, err := common.EncodeMessage(common.MessageCommand, commandMessage{
		ID:      id,
		Command: com,
	})
	if err != nil {
		return err
	}
	data, err := json.Marshal(com)
	if err != nil {
		return err
	}
	return updatePost(id, op, msg, "insertCommand", data)
}

// InsertLinks writes new links to other posts and the accompanying backlinks to
// the database
func InsertLinks(id, op uint64, links [][2]uint64) error {
	msg, err := common.EncodeMessage(common.MessageLink, linkMessage{
		ID:    id,
		Links: links,
	})
	if err != nil {
		return err
	}
	err = updatePost(id, op, msg, "insertLinks", linkRow(links))
	if err != nil {
		return err
	}

	// Most often this loop will iterate only once, so no need to think heavily
	// on optimizations
	for _, l := range links {
		bl := [][2]uint64{{id, op}}
		msg, err := common.EncodeMessage(common.MessageBacklink, linkMessage{
			ID:    l[0],
			Links: bl,
		})
		if err != nil {
			return err
		}
		err = updatePost(l[0], l[1], msg, "insertBacklinks", linkRow(bl))
		if err != nil {
			return err
		}
	}

	return nil
}

// Backspace removes one character from the end of the post body
func Backspace(id, op uint64) error {
	msg, err := common.EncodeMessage(common.MessageBackspace, id)
	if err != nil {
		return err
	}
	return updatePost(id, op, msg, "backspace", nil)
}

// ClosePost closes an open post
func ClosePost(id, op uint64) error {
	msg, err := common.EncodeMessage(common.MessageClosePost, id)
	if err != nil {
		return err
	}
	return updatePost(id, op, msg, "closePost", nil)
}

// InsertImage insert and image into and existing open post
func InsertImage(id, op uint64, img common.Image) (err error) {
	msg, err := common.EncodeMessage(common.MessageInsertImage, imageMessage{
		ID:    id,
		Image: img,
	})
	if err != nil {
		return
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer RollbackOnError(tx, &err)

	_, err = tx.Stmt(prepared["insertImage"]).Exec(id, img.SHA1, img.Name)
	if err != nil {
		return
	}
	err = BumpThread(tx, op, false, false, true, msg)
	if err != nil {
		return
	}
	return tx.Commit()
}

// SplicePost splices the text body of a post. For less load on the DB, supply
// the entire new body as `body`.
func SplicePost(id, op uint64, msg []byte, body string) error {
	return updatePost(id, op, msg, "replaceBody", body)
}
