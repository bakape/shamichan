package parser

import (
	"regexp"
	"strconv"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
)

var linkRegexp = regexp.MustCompile(`(?:^| )>{2,}(\d+)\b`)

// Extract post links from a text fragment, verify and retrieve their
// parenthood
func parseLinks(frag []byte) (types.LinkMap, error) {
	matches := linkRegexp.FindAllSubmatch(frag, -1)
	if matches == nil {
		return nil, nil
	}
	links := make(types.LinkMap, len(matches))
	for _, match := range matches {
		id, err := strconv.ParseInt(string(match[1]), 10, 64)
		if err != nil {
			return nil, err
		}

		var parent struct {
			ID    int64
			Board string
		}
		q := db.FindPost(id).Pluck("id", "board").Default(nil)
		err = db.One(q, &parent)
		if err != nil {
			if err == r.ErrEmptyResult { // Points to invalid post. Ignore.
				continue
			}
			return nil, err
		}
		links[id] = types.Link{
			OP:    parent.ID,
			Board: parent.Board,
		}
	}

	// All links invalid
	if len(links) == 0 {
		return nil, nil
	}

	return links, nil
}
