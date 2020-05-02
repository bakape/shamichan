package common

// #cgo pkg-config: libavformat
// #cgo CFLAGS: -std=c11 -g
// #include <libavformat/avformat.h>
import "C"
import (
	"errors"
	"fmt"
	"strings"

	"github.com/bakape/thumbnailer/v2"
	"nhooyr.io/websocket"
)

// Commonly used errors
var (
	ErrNameTooLong         = ErrTooLong("name")
	ErrSubjectTooLong      = ErrTooLong("subject")
	ErrPostPasswordTooLong = ErrTooLong("post password")
	ErrBodyTooLong         = ErrTooLong("post body")
	ErrContainsNull        = ErrInvalidInput("null byte in message")
	ErrInvalidCaptcha      = ErrInvalidInput("captcha")
	ErrTooManyConnections  = ErrAccessDenied("too many connections")
	ErrNoPermissions       = ErrAccessDenied("insufficient permissions")
)

// StatusError is a simple error with HTTP status code attached
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

// ErrInvalidInput is an error that invalid user input was supplied
func ErrInvalidInput(s string) error {
	return StatusError{errors.New(s), 400}
}

// Wrap and return any error returned by f as a StatusError with the passed code
func WrapError(code int, f func() (err error)) (err error) {
	err = f()
	if err != nil {
		err = StatusError{
			Err:  err,
			Code: code,
		}
	}
	return
}

// ErrAccessDenied is an error that user does not have enough access rights
func ErrAccessDenied(s string) error {
	return StatusError{errors.New(s), 403}
}

// ErrNonPrintable is an error that user input has non-printable runes
func ErrNonPrintable(r rune) error {
	return StatusError{
		fmt.Errorf("contains non-printable character: %d", int(r)),
		400,
	}
}

// ErrInvalidThread is an error that no such thread on this board
func ErrInvalidThread(id uint64, board string) error {
	return StatusError{
		fmt.Errorf("no thread %d on board `%s`", id, board),
		404,
	}
}

// ErrInvalidBoard is an error that an invalid board was provided
func ErrInvalidBoard(board string) error {
	return StatusError{fmt.Errorf("board `%s` does not exist", board), 404}
}

// Enum decoding error
func ErrInvalidEnum(s string) error {
	return StatusError{fmt.Errorf("invalid enum: %s", s), 400}
}

// CanIgnoreClientError returns, if client-caused error can be safely ignored and not logged
func CanIgnoreClientError(err error) bool {
	if err == nil {
		return true
	}

	switch err.(type) {
	case StatusError:
		err := err.(StatusError)
		c := err.Code
		if (c >= 400 && c < 500) ||
			strings.HasPrefix(err.Err.Error(), "YouTube") {
			return true
		}
	case thumbnailer.AVError:
		switch err.(thumbnailer.AVError).Code() {
		case C.AVERROR_INVALIDDATA, // Invalid uploaded data need not be logged
			C.AVERROR_EXTERNAL: // Not much can be done about unspecified errors
			return true
		default:
			return false
		}
	case websocket.CloseError:
		return true
	}

	err = errors.Unwrap(err)
	if err != nil {
		return CanIgnoreClientError(err)
	}
	return false
}
