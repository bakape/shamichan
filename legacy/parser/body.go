// Package parser parses and verifies user-sent post data
package parser

import (
	"regexp"
	"strconv"
	"unicode"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/util"
)

var (
	linkRegexp = regexp.MustCompile(`^>{2,}(\d+)$`)
)

// ParseBody parses the entire post text body for commands and links.
// internal: function was called by automated upkeep task
func ParseBody(body []byte, internal bool) (
	links []uint64,
	com []common.Command,
	err error,
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
			var id uint64
			id, err = strconv.ParseUint(string(m[1]), 10, 64)
			if err != nil {
				return
			}
			links = append(links, id)
		case '#':
			// Ignore hash commands in quotes, or #pyu/#pcount if board option disabled
			if body[lineStart] == '>' || (len(word) > 1 && word[1] == 'p') {
				goto next
			}
			m := common.CommandRegexp.FindSubmatch(word)
			if m == nil {
				goto next
			}
			var c common.Command
			c, err = parseCommand(m[1])
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
