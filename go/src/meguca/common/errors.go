package common

import (
	"errors"
	"strconv"
)

// Commonly used errors
var (
	ErrNameTooLong         = ErrTooLong("name")
	ErrSubjectTooLong      = ErrTooLong("subject")
	ErrPostPasswordTooLong = ErrTooLong("post password")
	ErrBodyTooLong         = ErrTooLong("post body")
	ErrInvalidCreds        = errors.New("invalid login credentials")
	ErrContainsNull        = errors.New("null byte in non-concatenated message")
)

// ErrTooLong is passed, when a field exceeds the maximum string length for
// that specific field
type ErrTooLong string

func (e ErrTooLong) Error() string {
	return string(e) + " too long"
}

// ErrInvalidPostID signifies that the post ID passed by the client is invalid
// in some way. In what way exactly should be evident from the API endpoint.
type ErrInvalidPostID uint64

func (e ErrInvalidPostID) Error() string {
	return "invalid post ID: " + strconv.FormatUint(uint64(e), 10)
}
