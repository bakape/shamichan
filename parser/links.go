package parser

import (
	"database/sql"
	"regexp"
	"strconv"
	"strings"

	"github.com/bakape/meguca/db"
)

var linkRegexp = regexp.MustCompile(`^>{2,}(\d+)$`)

// Extract post links from a text fragment, verify and retrieve their
// parenthood
func parseLinks(frag string) ([][2]uint64, error) {
	var links [][2]uint64

	// TODO: Do this in-place w/o creating any garbage slices
	for _, word := range strings.Split(frag, " ") {
		if len(word) == 0 || word[0] != '>' {
			continue
		}

		match := linkRegexp.FindStringSubmatch(word)
		if match == nil {
			continue
		}

		id, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil {
			return nil, err
		}

		op, err := db.GetPostOP(id)
		switch err {
		case nil:
			links = append(links, [2]uint64{id, op})
		case sql.ErrNoRows: // Points to invalid post. Ignore.
			continue
		default:
			return nil, err
		}
	}

	return links, nil
}
