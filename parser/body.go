package parser

import (
	"regexp"
	"strings"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
)

var (
	commandRegexp = regexp.MustCompile(`^#(flip|\d*d\d+|8ball)$`)

	errBodyTooLong = ErrTooLong("post body")
)

// BodyParser parses post text bodies or their fragments
type BodyParser struct {
	Config config.PostParseConfigs
	Board  string
}

// BodyParseResults stores the results of parsing a post body or its fragment
type BodyParseResults struct {
	Body     string
	Links    types.LinkMap
	Commands []types.Command
}

// ParseBody parses a full text body of a post
func (b BodyParser) ParseBody(body string) (res BodyParseResults, err error) {
	if len(body) > maxLengthBody {
		return res, errBodyTooLong
	}

	body = stripAndTrim(body)
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if b.Config.HashCommands {
			command := commandRegexp.FindStringSubmatch(line)
			if command != nil {
				res.Commands, err = b.parseCommand(res.Commands, command[1])
				if err != nil {
					return res, err
				}
			}
		}
	}

	return res, nil
}
