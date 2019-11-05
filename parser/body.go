// Package parser parses and verifies user-sent post data
package parser

import (
	"github.com/Chiiruno/meguca/common"
	"github.com/Chiiruno/meguca/config"
	"github.com/Chiiruno/meguca/util"
	"regexp"
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
	links []common.Link, com []common.Command, err error,
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

	start := 0
	lineStart := 0
	pyu := config.GetBoardConfigs(board).Pyu

	// Prevent link duplication
	haveLink := make(map[uint64]bool)
	// Prevent #pyu duplication
	isSlut := false

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

		_, word, _ := util.SplitPunctuation(body[start:i])
		start = i + 1
		if len(word) == 0 {
			goto next
		}

		switch word[0] {
		case '>':
			m := linkRegexp.FindSubmatch(word)
			if m == nil {
				goto next
			}
			var l common.Link
			l, err = parseLink(m)
			switch {
			case err != nil:
				return
			case l.ID != 0:
				if !haveLink[l.ID] {
					haveLink[l.ID] = true
					links = append(links, l)
				}
			}
		case '#':
			// Ignore hash commands in quotes, or #pyu/#pcount if board option disabled
			if body[lineStart] == '>' || (len(word) > 1 && word[1] == 'p' && !pyu) {
				goto next
			}
			m := common.CommandRegexp.FindSubmatch(word)
			if m == nil {
				goto next
			}
			var c common.Command
			c, err = parseCommand(m[1], board, thread, id, ip, &isSlut)
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
	next:
		if b == '\n' {
			lineStart = i + 1
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
