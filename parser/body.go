// Package parser parses and verifies user-sent post data
package parser

import (
	"bytes"
	"regexp"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
)

var (
	// CommandRegexp matches any hash command in a line
	CommandRegexp = regexp.MustCompile(`^#(flip|\d*d\d+|8ball|pyu|pcount)$`)
)

// ParseBody parses the entire post text body for commands and links
func ParseBody(body []byte, board string) (
	links [][2]uint64, com []common.Command, err error,
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
		for _, l := range l {
			links = append(links, l)
		}
	}

	return
}

func parseLine(line []byte, board string, parseCommands bool) (
	links [][2]uint64, com common.Command, err error,
) {
	if len(line) == 0 {
		return
	}

	if parseCommands && line[0] == '#' {
		if m := CommandRegexp.FindSubmatch(line); m != nil {
			com, err = parseCommand(string(m[1]), board)
			if err != nil {
				return
			}
		}
	}

	links, err = parseLinks(line)
	return
}
