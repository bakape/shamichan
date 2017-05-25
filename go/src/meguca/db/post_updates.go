package db

import (
	"meguca/common"
)

// ClosePost closes an open post and commits any links and hash commands
func ClosePost(id, op uint64, body string, links [][2]uint64, com []common.Command) (
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
		return err
	}

	err = execPrepared("close_post", id, body, linkRow(links), commandRow(com))
	if err != nil {
		return
	}

	if !IsTest {
		common.ClosePost(id, op, msg)
	}
	return deleteOpenPostBody(id)
}
