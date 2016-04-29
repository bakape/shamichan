// Package util contains various general utility functions used throughout
// the project.
package util

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"time"
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
func HashBuffer(buf []byte) (string, error) {
	hash := md5.Sum(buf)
	return hex.EncodeToString(hash[:])[:16], nil
}

// CopyFile reads a file from disk and copies it into the writer
func CopyFile(path string, writer io.Writer) error {
	file, err := os.Open(path)
	if err != nil {
		return copyError(err)
	}
	defer file.Close()
	_, err = io.Copy(writer, file)
	if err != nil {
		return copyError(err)
	}
	return nil
}

func copyError(err error) error {
	return WrapError("Error copying file", err)
}

// IDToString is a  helper for converting a post ID to a string for JSON keys
func IDToString(id uint64) string {
	return strconv.FormatUint(id, 10)
}

// LogError logs an error with its stack trace
func LogError(ip string, err error) {
	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	log.Printf("panic serving %v: %v\n%s", ip, err, buf)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

const randSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"0123456789"

// RandomID generates a randomID of uppercase and lowercase letters and numbers
// of desired length
func RandomID(length int) string {
	buf := make([]byte, length)
	for i := range buf {
		buf[i] = randSource[rand.Int63()%int64(len(randSource))]
	}
	return string(buf)
}
