package db

import (
	"database/sql"
	"meguca/common"
)

// ClosePost closes an open post and commits any links and hash commands
func ClosePost(id, op uint64, body string, links []common.Link, com []common.Command) (
	err error,
) {
	msg, err := common.EncodeMessage(common.MessageClosePost, struct {
		ID       uint64           `json:"id"`
		Links    []common.Link    `json:"links,omitempty"`
		Commands []common.Command `json:"commands,omitempty"`
	}{
		ID:       id,
		Links:    links,
		Commands: com,
	})
	if err != nil {
		return
	}

	err = InTransaction(false, func(tx *sql.Tx) (err error) {
		q := sq.Update("posts").
			SetMap(map[string]interface{}{
				"editing":  false,
				"body":     body,
				"commands": commandRow(com),
				"password": nil,
			}).
			Where("id = ?", id)
		err = withTransaction(tx, q).Exec()
		if err != nil {
			return
		}
		err = writeLinks(tx, id, links)
		if err != nil {
			return
		}
		err = bumpThread(tx, op, false)
		return
	})
	if err != nil {
		return
	}

	if !IsTest {
		common.ClosePost(id, op, links, com, msg)
	}
	return deleteOpenPostBody(id)
}
