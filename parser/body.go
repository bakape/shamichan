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

	// Find and parse hash commands
	if b.Config.HashCommands {
		for _, line := range strings.Split(body, "\n") {
			match := commandRegexp.FindStringSubmatch(line)
			if match != nil {
				res.Commands, err = b.parseCommand(res.Commands, match[1])
				if err != nil {
					return res, err
				}
			}
		}
	}

	res.Links, err = parseLinks(body)
	return
}
