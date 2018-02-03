// Package parser parses and verifies user-sent post data
package parser

import (
	"fmt"
	"meguca/common"
	"meguca/util"
	"regexp"
	"unicode"
)

var (
	linkRegexp = regexp.MustCompile(`^>{2,}(\d+)$`)
)

// Rune is non-printable
type ErrNonPrintable rune

func (e ErrNonPrintable) Error() string {
	return fmt.Sprintf("contains non-printable character: %d", int(e))
}

// Needed to avoid cyclic imports for the 'db' package
func init() {
	common.ParseBody = ParseBody
}

// ParseBody parses the entire post text body for commands and links
func ParseBody(body []byte, board string) (
	links [][2]uint64, com []common.Command, err error,
) {
	err = IsPrintableString(string(body), true)
	if err != nil {
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
func IsPrintable(r rune, multiline bool) error {
	switch r {
	case '\t', '\n':
		if !multiline {
			return ErrNonPrintable(r)
		}
	default:
		if !unicode.IsPrint(r) {
			return ErrNonPrintable(r)
		}
	}
	return nil
}

// Checks, if all of s is printable.
// Also accepts tabs, and newlines, if multiline = true.
func IsPrintableString(s string, multiline bool) error {
	for _, r := range []rune(s) {
		if err := IsPrintable(r, multiline); err != nil {
			return err
		}
	}
	return nil
}

// Checks, if all of s is printable.
// Also accepts tabs, and newlines, if multiline = true.
func IsPrintableRunes(s []rune, multiline bool) error {
	for _, r := range s {
		if err := IsPrintable(r, multiline); err != nil {
			return err
		}
	}
	return nil
}
