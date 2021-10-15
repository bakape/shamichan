// Package parser parses and verifies user-sent post data
package parser

import (
	"bytes"
	"fmt"
	"regexp"
	"sync"
	"unicode"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
)

var (
	filters   = make(map[uint64]*threadFilters)
	filtersMu sync.Mutex
)

type threadFilters struct {
	filters map[string]filter
	sync.RWMutex
}

type filter struct {
	re                 *regexp.Regexp
	to, toFirstCapital []byte
}

var (
	linkRegexp = regexp.MustCompile(`^>{2,}(\d+)$`)
	filterRe   = regexp.MustCompile(`^#filter ([\w\s]+) -> ([\w\s\d#\(\)]+)$`)
)

// Needed to avoid cyclic imports for the 'db' package
func init() {
	common.ParseBody = ParseBody
}

func registerFilter(thread uint64, body []byte) {
outer:
	for _, l := range bytes.Split(body, []byte("\n")) {
		m := filterRe.FindSubmatch(l)
		if m == nil {
			continue
		}
		if m == nil {
			continue
		}

		from := bytes.ToLower(bytes.TrimSpace(m[1]))
		to := bytes.ToLower(bytes.TrimSpace(m[2]))
		for _, w := range [...][]byte{from, to} {
			if len(w) < 2 || len(w) > 20 {
				continue outer
			}
		}

		filtersMu.Lock()
		f := filters[thread]
		if f == nil {
			f = &threadFilters{
				filters: make(map[string]filter),
			}
			filters[thread] = f
		}
		filtersMu.Unlock()

		var firstCapital []byte
		if to[0] >= 'a' && to[0] <= 'z' {
			firstCapital = append(firstCapital, to...)
			firstCapital[0] -= 'a' - 'A'
		} else {
			firstCapital = to
		}

		re := regexp.MustCompile(fmt.Sprintf(`(?i)\b%s\b`, from))

		f.Lock()
		f.filters[string(from)] = filter{
			re:             re,
			to:             to,
			toFirstCapital: firstCapital,
		}
		f.Unlock()

		break
	}
}

func ApplyFilters(thread uint64, body *[]byte) (applied bool) {
	filtersMu.Lock()
	f := filters[thread]
	filtersMu.Unlock()

	if f == nil {
		return
	}
	f.RLock()
	defer f.RUnlock()

	for _, p := range f.filters {
		*body = p.re.ReplaceAllFunc(*body, func(b []byte) []byte {
			applied = true

			// Minor capitalization support
			if b[0] >= 'A' && b[0] <= 'Z' {
				return p.toFirstCapital
			} else {
				return p.to
			}
		})
	}

	if applied {
		*body = bytes.ReplaceAll(
			*body,
			[]byte("#autobahn"),
			[]byte(`@@^rthere ain't no rest for the wicked^r@@`),
		)
	}
	if len(*body) > 2000 {
		*body = (*body)[:2000]
	}
	return
}

// ParseBody parses the entire post text body for commands and links.
// internal: function was called by automated upkeep task
func ParseBody(
	body []byte,
	board string,
	thread uint64,
	id uint64,
	ip string,
	internal bool,
) (
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
	// Prevent #autobahn duplication
	isDead := false

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
			if body[lineStart] == '>' ||
				(len(word) > 1 && word[1] == 'p' && !pyu) {
				goto next
			}
			m := common.CommandRegexp.FindSubmatch(word)
			if m == nil {
				goto next
			}
			var c common.Command
			c, err = parseCommand(m[1], board, thread, id, ip, &isSlut, &isDead)
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

	registerFilter(thread, body)

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
