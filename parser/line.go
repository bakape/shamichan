package parser

import (
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

// ParseLine parses a full text line of a post
func ParseLine(line []byte, board string) (
	links types.LinkMap, command types.Command, err error,
) {
	// Find and parse hash commands
	if config.GetBoardConfigs(board).HashCommands {
		match := CommandRegexp.FindSubmatch(line)
		if match != nil {
			command, err = parseCommand(match[1], board)
			return
		}
	}

	links, err = parseLinks(line)
	return
}
