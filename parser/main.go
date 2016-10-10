// Package parser parses and verifies user-sent post data
package parser

const (
	maxLengthName         = 50
	maxLengthEmail        = 100
	maxLengthAuth         = 50
	maxLengthPostPassword = 50
	maxLengthSubject      = 100

	// MaxLengthBody is the maximum allowed length of a post text body
	MaxLengthBody = 2000
)

// ErrTooLong is passed, when a field exceeds the maximum string lenghth for
// that specific field
type ErrTooLong string

func (e ErrTooLong) Error() string {
	return string(e) + " too long"
}
