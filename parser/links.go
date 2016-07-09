package parser

import (
	"regexp"
	"strconv"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
)

var linkRegexp = regexp.MustCompile(`>>(\d+)`)

// Extract post links from a text fragment, verify and retrieve their
// parenthood
func parseLinks(frag string) (types.LinkMap, error) {
	matches := linkRegexp.FindAllStringSubmatch(frag, -1)
	if matches == nil {
		return nil, nil
	}
	links := make(types.LinkMap, len(matches))
	for _, match := range matches {
		id, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return nil, err
		}

		var link types.Link
		err = db.One(db.FindPost(id).Pluck("op", "board").Default(nil), &link)
		if err != nil {
			if err == r.ErrEmptyResult { // Points to invalid post. Ignore.
				continue
			}
			return nil, err
		}
		links[id] = link
	}

	// All links invalid
	if len(links) == 0 {
		return nil, nil
	}

	return links, nil
}
