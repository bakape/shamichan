// Package parser parses and verifies user-sent post data
package parser

import (
	"regexp"
	"strings"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
)

var (
	// CommandRegexp matches any hash command in a line
	CommandRegexp = regexp.MustCompile(`^#(flip|\d*d\d+|8ball|pyu|pcount)$`)
)

// ParseBody parses the entire post text body for commands and links
func ParseBody(body, board string) (
	links [][2]uint64, com []common.Command, err error,
) {
	parseCommands := config.GetBoardConfigs(board).HashCommands
	for _, line := range strings.Split(body, "\n") {
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

// ParseLine parses a full text line of a post
func ParseLine(line, board string) ([][2]uint64, common.Command, error) {
	return parseLine(line, board, config.GetBoardConfigs(board).HashCommands)
}

func parseLine(line, board string, parseCommands bool) (
	links [][2]uint64, com common.Command, err error,
) {
	if len(line) == 0 {
		return
	}

	if parseCommands && line[0] == '#' {
		if m := CommandRegexp.FindStringSubmatch(line); m != nil {
			com, err = parseCommand(m[1], board)
			if err != nil {
				return
			}
		}
	}

	links, err = parseLinks(line)
	return
}
