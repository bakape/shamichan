// Package util contains various general utility functions used throughout
// the project.
package util

import (
	"crypto/md5"
	"encoding/hex"
	"log"
	"runtime"
	"strconv"
)

// WrapError wraps error types to create compound error chains
func WrapError(text string, err error) error {
	return wrapedError{
		text:  text,
		inner: err,
	}
}

type wrapedError struct {
	text  string
	inner error
}

func (e wrapedError) Error() string {
	text := e.text
	if e.inner != nil {
		text += ": " + e.inner.Error()
	}
	return text
}

// Waterfall executes a slice of functions until the first error returned. This
// error, if any, is returned to the caller.
func Waterfall(fns []func() error) (err error) {
	for _, fn := range fns {
		err = fn()
		if err != nil {
			break
		}
	}
	return
}

// HashBuffer computes a truncated MD5 hash from a buffer
func HashBuffer(buf []byte) string {
	hash := md5.Sum(buf)
	return hex.EncodeToString(hash[:])[:16]
}

// IDToString is a  helper for converting a post ID to a string for JSON keys
func IDToString(id int64) string {
	return strconv.FormatInt(id, 10)
}

// LogError logs an error with its stack trace
func LogError(ip string, err interface{}) {
	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	log.Printf("panic serving %v: %v\n%s", ip, err, buf)
}
