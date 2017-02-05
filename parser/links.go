package parser

import (
	"database/sql"
	"strconv"

	"github.com/bakape/meguca/db"
)

// Extract post links from a text fragment, verify and retrieve their
// parenthood
func parseLink(match [][]byte) (link [2]uint64, err error) {
	id, err := strconv.ParseUint(string(match[1]), 10, 64)
	if err != nil {
		return
	}

	op, err := db.GetPostOP(id)
	switch err {
	case nil:
		link = [2]uint64{id, op}
	case sql.ErrNoRows: // Points to invalid post. Ignore.
		err = nil
	}
	return
}
