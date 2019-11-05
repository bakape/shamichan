package parser

import (
	"database/sql"
	"github.com/Chiiruno/meguca/common"
	"strconv"

	"github.com/Chiiruno/meguca/db"
)

// Extract post links from a text fragment, verify and retrieve their
// parenthood
func parseLink(match [][]byte) (link common.Link, err error) {
	id, err := strconv.ParseUint(string(match[1]), 10, 64)
	if err != nil {
		return
	}

	board, op, err := db.GetPostParenthood(id)
	switch err {
	case nil:
		link = common.Link{
			ID:    id,
			OP:    op,
			Board: board,
		}
	case sql.ErrNoRows: // Points to invalid post. Ignore.
		err = nil
	}
	return
}
