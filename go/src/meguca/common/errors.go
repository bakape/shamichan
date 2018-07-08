package common

import (
	"errors"
	"fmt"
	"meguca/util"
	"strings"

	"github.com/gorilla/websocket"
)

// Commonly used errors
var (
	ErrNameTooLong         = ErrTooLong("name")
	ErrSubjectTooLong      = ErrTooLong("subject")
	ErrPostPasswordTooLong = ErrTooLong("post password")
	ErrBodyTooLong         = ErrTooLong("post body")
	ErrContainsNull        = ErrInvalidInput("null byte in message")
	ErrInvalidCaptcha      = ErrInvalidInput("captcha")
	ErrInvalidCreds        = ErrAccessDenied("login credentials")
	ErrBanned              = ErrAccessDenied("you are banned from this board")

	// The poster is almost certainly spamming
	ErrSpamDected = ErrAccessDenied("spam detected")
)

// Simple error with HTTP status code attached
type StatusError struct {
	Err  error
	Code int
}

func (e StatusError) Error() string {
	var prefix string
	switch e.Code {
	case 400:
		prefix = "invalid input"
	case 403:
		prefix = "access denied"
	case 404:
		prefix = "not found"
	case 500:
		prefix = "internal server error"
	}
	return fmt.Sprintf("%s: %s", prefix, e.Err)
}

// ErrTooLong is passed, when a field exceeds the maximum string length for
// that specific field
func ErrTooLong(s string) error {
	return StatusError{errors.New(s + " too long"), 400}
}

// Invalid user input was suplied
func ErrInvalidInput(s string) error {
	return StatusError{errors.New(s), 400}
}

// User does not have enough access rights
func ErrAccessDenied(s string) error {
	return StatusError{errors.New(s), 403}
}

// User input has non-printable runes
func ErrNonPrintable(r rune) error {
	return StatusError{
		fmt.Errorf("contains non-printable character: %d", int(r)),
		400,
	}
}

// No such thread on this board
func ErrInvalidThread(id uint64, board string) error {
	return StatusError{
		fmt.Errorf("no thread %d on board `%s`", id, board),
		404,
	}
}

// Invalid board provided
func ErrInvalidBoard(board string) error {
	return StatusError{fmt.Errorf("board `%s` does not exist", board), 404}
}

// Returns, if client-caused error can be safely ignored and not logged
func CanIgnoreClientError(err error) bool {
recheck:
	switch err {
	case websocket.ErrCloseSent, nil:
		return true
	}

	switch err.(type) {
	case StatusError:
		if c := err.(StatusError).Code; c >= 400 && c < 500 {
			return true
		}
	case util.WrappedError:
		err = err.(util.WrappedError).Inner
		goto recheck
	}

	// Ignore client-side connection loss
	s := err.Error()
	for _, suff := range [...]string{
		"connection reset by peer",
		"broken pipe",
	} {
		if strings.HasSuffix(s, suff) {
			return true
		}
	}

	return false
}
