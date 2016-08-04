// Package parser parses and verifies user-sent post data
package parser

import (
	"bytes"
	"strings"
)

const (
	maxLengthName         = 50
	maxLengthEMail        = 100
	maxLengthAuth         = 50
	maxLengthPostPassword = 50
	maxLengthSubject      = 100
	maxLengthBody         = 2000
)

// ErrTooLong is passed, when a field exceeds the maximum string lenghth for
// that specific field
type ErrTooLong string

func (e ErrTooLong) Error() string {
	return string(e) + " too long"
}

func stripAndTrim(s string) string {
	return strings.TrimSpace(stripPsuedoWhitespace(s))
}

// Strip white-space like unicode characters from srings to avoid "faking"
// spaces
func stripPsuedoWhitespace(s string) string {
	buf := bytes.NewBuffer(make([]byte, 0, len(s)))
	for _, r := range s {
		if r >= 0x2000 && r <= 0x206f {
			if r <= 0x200f || r >= 0x205f || (r >= 0x202a && r <= 0x202f) {
				continue
			}
		}
		buf.WriteRune(r)
	}

	return buf.String()
}
