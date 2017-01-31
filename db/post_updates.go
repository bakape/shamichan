package db

import (
	"database/sql"
	"encoding/json"

	"github.com/bakape/meguca/common"
	"github.com/lib/pq"
)

// UpdateLog writes to a thread's replication log..
func UpdateLog(tx *sql.Tx, id uint64, msg []byte) error {
	_, err := getStatement(tx, "update_log").Exec(id, msg)
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
	return updatePost(id, op, msg, "append_body", string(char))
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
	err = updatePostTx(tx, id, op, msg, queryKey, arg)
	if err != nil {
		tx.Rollback()
		return
	}
	return tx.Commit()
}

func updatePostTx(
	tx *sql.Tx,
	id, op uint64,
	msg []byte,
	queryKey string,
	arg interface{},
) (err error) {
	q := tx.Stmt(prepared[queryKey])
	if arg != nil {
		_, err = q.Exec(id, arg)
	} else {
		_, err = q.Exec(id)
	}
	if err != nil {
		return
	}
	return UpdateLog(tx, op, msg)
}

// InsertCommand inserts a has command result into a post
func insertCommands(tx *sql.Tx, id, op uint64, com []common.Command) (
	err error,
) {
	data := make([]string, len(com))
	for i := range com {
		var b []byte
		b, err = json.Marshal(com[i])
		if err != nil {
			return
		}
		data[i] = string(b)
	}

	_, err = tx.Stmt(prepared["insert_commands"]).Exec(id, pq.StringArray(data))
	return
}

// Writes new links to other posts and the accompanying backlinks
func insertLinks(tx *sql.Tx, id, op uint64, links [][2]uint64) (err error) {
	_, err = tx.Stmt(prepared["insert_links"]).Exec(id, linkRow(links))
	if err != nil {
		return
	}

	// Most often this loop will iterate only once, so no need to think heavily
	// on optimizations
	for _, l := range links {
		var msg []byte
		msg, err = common.EncodeMessage(
			common.MessageBacklink,
			[3]uint64{l[0], id, op},
		)
		if err != nil {
			return
		}
		err = updatePostTx(
			tx,
			l[0],
			l[1],
			msg,
			"insert_backlinks",
			linkRow{{id, op}},
		)
		if err != nil {
			return
		}
	}

	return
}

// Backspace removes one character from the end of the post body
func Backspace(id, op uint64) error {
	msg, err := common.EncodeMessage(common.MessageBackspace, id)
	if err != nil {
		return err
	}
	return updatePost(id, op, msg, "backspace", nil)
}

// ClosePost closes an open post and commits any links, backlinks and hash
// commands
func ClosePost(id, op uint64, links [][2]uint64, com []common.Command) (
	err error,
) {
	msg, err := common.EncodeMessage(common.MessageClosePost, struct {
		ID       uint64           `json:"id"`
		Links    [][2]uint64      `json:"links,omitempty"`
		Commands []common.Command `json:"commands,omitempty"`
	}{
		ID:       id,
		Links:    links,
		Commands: com,
	})
	if err != nil {
		return
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer RollbackOnError(tx, &err)

	err = updatePostTx(tx, id, op, msg, "close_post", nil)
	if err != nil {
		return
	}

	if com != nil {
		err = insertCommands(tx, id, op, com)
		if err != nil {
			return
		}
	}
	if links != nil {
		err = insertLinks(tx, id, op, links)
		if err != nil {
			return
		}
	}

	return tx.Commit()
}

// InsertImage insert and image into and existing open post
func InsertImage(id, op uint64, img common.Image) (err error) {
	msg, err := common.EncodeMessage(common.MessageInsertImage, struct {
		common.Image
		ID uint64 `json:"id"`
	}{
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

	_, err = tx.Stmt(prepared["insert_image"]).Exec(id, img.SHA1, img.Name)
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
	return updatePost(id, op, msg, "replace_body", body)
}
