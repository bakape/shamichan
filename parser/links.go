package parser

import (
	"bytes"
	"regexp"
	"strconv"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
)

var linkRegexp = regexp.MustCompile(`^>{2,}(\d+)$`)

// Extract post links from a text fragment, verify and retrieve their
// parenthood
func parseLinks(frag []byte) (common.LinkMap, error) {
	var links common.LinkMap

	// TODO: Do this in-place w/o creating any garbage slices
	for _, word := range bytes.Split(frag, []byte{' '}) {
		if len(word) == 0 || word[0] != '>' {
			continue
		}

		match := linkRegexp.FindSubmatch(word)
		if match == nil {
			continue
		}

		id, err := strconv.ParseUint(string(match[1]), 10, 64)
		if err != nil {
			return nil, err
		}

		var parenthood struct {
			OP    uint64
			Board string
		}
		q := db.FindPost(id).Pluck("op", "board").Default(nil)
		err = db.One(q, &parenthood)
		if err != nil {
			if err == r.ErrEmptyResult { // Points to invalid post. Ignore.
				continue
			}
			return nil, err
		}

		link := common.Link{
			OP:    parenthood.OP,
			Board: parenthood.Board,
		}

		if links == nil {
			links = common.LinkMap{id: link}
		} else {
			links[id] = link
		}
	}

	// All links invalid
	if len(links) == 0 {
		return nil, nil
	}

	return links, nil
}
