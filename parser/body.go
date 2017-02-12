// Package parser parses and verifies user-sent post data
package parser

import (
	"regexp"

	"github.com/bakape/meguca/common"
)

// Post syntax pattern matchers
var (
	CommandRegexp = regexp.MustCompile(`^#(flip|\d*d\d+|8ball|pyu|pcount)$`)
	linkRegexp    = regexp.MustCompile(`^>{2,}(\d+)$`)
)

// Needed to avoid cyclic imports for the 'db' package
func init() {
	common.ParseBody = ParseBody
}

// ParseBody parses the entire post text body for commands and links
func ParseBody(body []byte, board string) (
	links [][2]uint64, com []common.Command, err error,
) {
	start := 0

	for i, b := range body {
		switch b {
		case '\n', ' ', '\t':
		default:
			if i == len(body)-1 {
				i++
			} else {
				continue
			}
		}

		word := body[start:i]
		start = i + 1
		if len(word) == 0 {
			continue
		}

		switch word[0] {
		case '>':
			m := linkRegexp.FindSubmatch(word)
			if m == nil {
				continue
			}
			var l [2]uint64
			l, err = parseLink(m)
			switch {
			case err != nil:
				return
			case l[0] != 0:
				links = append(links, l)
			}
		case '#':
			m := CommandRegexp.FindSubmatch(word)
			if m == nil {
				continue
			}
			var c common.Command
			c, err = parseCommand(string(m[1]), board)
			switch {
			case err != nil:
				return
			case c.Val != nil:
				com = append(com, c)
			}
		}
	}

	return
}
