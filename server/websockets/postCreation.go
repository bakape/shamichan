package websockets

import (
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
)

var (
	errReadOnly = errInvalidMessage("read only board")
)

// Insert a new thread into the database
func insertThread(data []byte, c *Client) error {
	var req types.ThreadCreationRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}
	if !auth.IsNonMetaBoard(req.Board) {
		return errInvalidBoard
	}

	var conf config.PostParseConfigs
	if err := db.One(db.GetBoardConfig(req.Board), &conf); err != nil {
		return err
	}
	if conf.ReadOnly {
		return errReadOnly
	}

	// TODO: Thread creation cooldown

	return nil
}
