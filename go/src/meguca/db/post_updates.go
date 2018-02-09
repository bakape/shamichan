package db

import (
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

	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer RollbackOnError(tx, &err)
	_, err = tx.Stmt(prepared["close_post"]).Exec(id, body, commandRow(com))
	if err != nil {
		return
	}
	err = writeLinks(tx, id, links)
	if err != nil {
		return
	}
	err = tx.Commit()
	if err != nil {
		return
	}

	if !IsTest {
		common.ClosePost(id, op, links, com, msg)
	}
	return deleteOpenPostBody(id)
}
