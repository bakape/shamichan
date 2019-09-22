// Package parser parses and verifies user-sent post data
package parser

import (
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	"github.com/bakape/meguca/websockets/feeds"
	"regexp"
	"strings"
	"unicode"
)

var (
	linkRegexp = regexp.MustCompile(`^>{2,}(\d+)$`)
)

// Needed to avoid cyclic imports for the 'db' package
func init() {
	common.ParseBody = ParseBody
}

// ParseBody parses the entire post text body for commands and links.
// internal: function was called by automated upkeep task
func ParseBody(body []byte, board string, thread uint64, id uint64, ip string, internal bool) (
	links []common.Link, com []common.Command, spamScore uint, err error,
) {
	err = IsPrintableString(string(body), true)
	if err != nil {
		if internal {
			err = nil
			// Strip any non-printables for automated post closing
			s := make([]byte, 0, len(body))
			for _, r := range []rune(string(body)) {
				if IsPrintable(r, true) == nil {
					s = append(s, string(r)...)
				}
			}
			body = s
		} else {
			return
		}
	}

	bodyStr := strings.Trim(string(body), "\n")
	boardConfig := config.GetBoardConfigs(board)

	// Prevent link duplication
	haveLink := make(map[uint64]bool)
	// Prevent #pyu duplication
	isSlut := false
	// Prevent many cinema pushes in one post
	pushedAlready := false

	for _, line := range strings.Split(bodyStr, "\n") {
		if boardConfig.CinemaEnabled && !internal {
			if !pushedAlready {
				if match := common.GetCinemaPushRegexp().FindStringSubmatch(line) ;
				match != nil {
					feeds.PushToCinema(thread, match[1], ip)
					spamScore = config.Get().CinemaPushScore
					pushedAlready = true
				}
			}
			if common.GetCinemaSkipRegexp().MatchString(line) {
				feeds.CinemaVoteSkip(thread, ip)
			}
		}
		for _, word := range strings.FieldsFunc(line, util.IsWordDelimiter) {
			if match := linkRegexp.FindStringSubmatch(word) ; match != nil {
				var l common.Link
				l, err = parseLink(match[1])
				switch {
				case err != nil:
					return
				case l.ID != 0:
					if !haveLink[l.ID] {
						haveLink[l.ID] = true
						links = append(links, l)
					}
				}
			} else if match := common.CommandRegexp.FindStringSubmatch(word) ; match != nil {
				// Ignore hash commands in quotes, or #pyu/#pcount if board option disabled
				if line[0] == '>' || (len(word) > 1 && word[1] == 'p' && !boardConfig.Pyu) {
					continue
				}
				var c common.Command
				c, err = parseCommand(match[1], board, thread, id, ip, &isSlut)
				switch err {
				case nil:
					com = append(com, c)
				case errTooManyRolls, errDieTooBig:
					// Consider command invalid
					err = nil
				default:
					return
				}
			}
		}

	}
	return
}

// IsPrintable checks, if r is printable.
// Also accepts tabs, and newlines, if multiline = true.
func IsPrintable(r rune, multiline bool) error {
	switch r {
	case '\t', '\n', 12288: // Japanese space
		if !multiline {
			return common.ErrNonPrintable(r)
		}
	default:
		if !unicode.IsPrint(r) {
			return common.ErrNonPrintable(r)
		}
	}
	return nil
}

// IsPrintableString checks, if all of s is printable.
// Also accepts tabs, and newlines, if multiline = true.
func IsPrintableString(s string, multiline bool) error {
	for _, r := range []rune(s) {
		if err := IsPrintable(r, multiline); err != nil {
			return err
		}
	}
	return nil
}

// IsPrintableRunes checks, if all of s is printable.
// Also accepts tabs, and newlines, if multiline = true.
func IsPrintableRunes(s []rune, multiline bool) error {
	for _, r := range s {
		if err := IsPrintable(r, multiline); err != nil {
			return err
		}
	}
	return nil
}
