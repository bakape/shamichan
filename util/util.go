// Package util contains various general utility functions used throughout
// the project.
package util

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"

	"golang.org/x/crypto/bcrypt"
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

// RandomID generates a randomID of bas64 characters of desired byte length
func RandomID(length int) (string, error) {
	buf := make([]byte, length)
	_, err := rand.Read(buf)
	return base64.RawStdEncoding.EncodeToString(buf), err
}

// PasswordHash generates a bcrypt hash from the passed login ID and password
func PasswordHash(id, password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(id+password), 10)
}

// ComparePassword conpares a bcrypt hash with the login ID and password of a
// user
func ComparePassword(id, password string, hash []byte) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(id+password))
}
