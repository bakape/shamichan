// Package parser parses and verifies user-sent post data
package parser

import (
	"errors"
	"meguca/common"
	"meguca/util"
	"regexp"
	"unicode"
)

var (
	linkRegexp = regexp.MustCompile(`^>{2,}(\d+)$`)

	// String or rune contains nonprintable character
	ErrContainsNonPrintable = errors.New("contains non-printable characters")
)

// Needed to avoid cyclic imports for the 'db' package
func init() {
	common.ParseBody = ParseBody
}

// ParseBody parses the entire post text body for commands and links
func ParseBody(body []byte, board string) (
	links [][2]uint64, com []common.Command, err error,
) {
	if !IsPrintableString(string(body), true) {
		err = ErrContainsNonPrintable
		return
	}

	start := 0

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
			continue
		}

		switch word[0] {
		case '>':
			m := linkRegexp.FindSubmatch(word)
			if m == nil {
				continue
			}
			var l [2]uint64
			l, err = parseLink(m)
			switch {
			case err != nil:
				return
			case l[0] != 0:
				links = append(links, l)
			}
		case '#':
			m := common.CommandRegexp.FindSubmatch(word)
			if m == nil {
				continue
			}
			var c common.Command
			c, err = parseCommand(m[1], board)
			switch err {
			case nil:
				com = append(com, c)
			case errTooManyRolls, errDieTooBig: // Consider command invalid
				err = nil
			default:
				return
			}
		}
	}

	return
}

// Checks, if r is printable.
// Also accepts tabs, and newlines, if multiline = true.
func IsPrintable(r rune, multiline bool) bool {
	switch r {
	case '\t', '\n':
		return multiline
	default:
		return unicode.IsPrint(r)
	}
}

// Checks, if all of s is printable.
// Also accepts tabs, and newlines, if multiline = true.
func IsPrintableString(s string, multiline bool) bool {
	for _, r := range []rune(s) {
		if !IsPrintable(r, multiline) {
			return false
		}
	}
	return true
}

// Checks, if all of s is printable.
// Also accepts tabs, and newlines, if multiline = true.
func IsPrintableRunes(s []rune, multiline bool) bool {
	for _, r := range s {
		if !IsPrintable(r, multiline) {
			return false
		}
	}
	return true
}
