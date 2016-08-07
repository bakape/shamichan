package parser

import (
	"regexp"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
)

var (
	commandRegexp = regexp.MustCompile(`^#(flip|\d*d\d+|8ball)$`)

	// ErrBodyTooLong is returned, when a post text body has exceeded
	// MaxLengthBody
	ErrBodyTooLong = ErrTooLong("post body")
)

// ParseLine parses a full text line of a post
func ParseLine(line []byte, board string) (
	links types.LinkMap, command types.Command, err error,
) {
	// Find and parse hash commands
	var enabled bool
	err = db.One(db.GetBoardConfig(board).Field("hashCommands"), &enabled)
	if err != nil {
		return
	}
	if enabled {
		match := commandRegexp.FindSubmatch(line)
		if match != nil {
			command, err = parseCommand(match[1], board)
			return
		}
	}

	links, err = parseLinks(line)
	return
}
