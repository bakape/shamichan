package parser

import (
	"bytes"
	"regexp"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
)

var (
	// CommandRegexp matches any hash command in a line
	CommandRegexp = regexp.MustCompile(`^#(flip|\d*d\d+|8ball|pyu|pcount)$`)

	// ErrBodyTooLong is returned, when a post text body has exceeded
	// MaxLengthBody
	ErrBodyTooLong = ErrTooLong("post body")
)

// ParseBody parses the entire post text body for commands and links
func ParseBody(body []byte, board string) (
	links types.LinkMap, com []types.Command, err error,
) {
	parseCommands := config.GetBoardConfigs(board).HashCommands
	for _, line := range bytes.Split(body, []byte{'\n'}) {
		l, c, err := parseLine(line, board, parseCommands)
		if err != nil {
			return nil, nil, err
		}
		if c.Val != nil {
			com = append(com, c)
		}
		if l != nil {
			if links == nil {
				links = l
			} else {
				for id, link := range l {
					links[id] = link
				}
			}
		}
	}

	return
}

// ParseLine parses a full text line of a post
func ParseLine(line []byte, board string) (
	types.LinkMap, types.Command, error,
) {
	return parseLine(line, board, config.GetBoardConfigs(board).HashCommands)
}

func parseLine(line []byte, board string, parseCommands bool) (
	links types.LinkMap, com types.Command, err error,
) {
	if len(line) == 0 {
		return
	}

	if parseCommands && line[0] == '#' {
		if m := CommandRegexp.FindSubmatch(line); m != nil {
			com, err = parseCommand(m[1], board)
			if err != nil {
				return
			}
		}
	}

	links, err = parseLinks(line)
	return
}
