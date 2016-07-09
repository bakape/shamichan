package parser

import (
	"errors"
	"regexp"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
)

var (
	commandRegexp       = regexp.MustCompile(`(?m)^#(flip|\d*d\d+|8ball)$`)
	allWhitespaceRegexp = regexp.MustCompile(`(?m)^\s*$`)

	errBodyTooLong = ErrTooLong("post body")

	// ErrOnlyWhitespace indicates the text body contains only whitespace and
	// therefore is invalid
	ErrOnlyWhitespace = errors.New("only whitespace in post body")
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
	if allWhitespaceRegexp.MatchString(body) {
		return res, ErrOnlyWhitespace
	}

	// Find and parse hash commands
	if b.Config.HashCommands {
		matches := commandRegexp.FindAllStringSubmatch(body, -1)
		if matches != nil {
			for _, match := range matches {
				res.Commands, err = b.parseCommand(res.Commands, match[1])
				if err != nil {
					return res, err
				}
			}
		}
	}

	res.Body = body
	res.Links, err = parseLinks(body)
	return
}
