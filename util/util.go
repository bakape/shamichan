// Package util contains various general utility functions used throughout
// the project.
package util

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

// WrapError wraps error types to create compound error chains
type WrapError struct {
	Text  string
	Inner error
}

// Error recursively converts the error chain into a string
func (e WrapError) Error() string {
	text := e.Text
	if e.Inner != nil {
		text += ": " + e.Inner.Error()
	}
	return text
}

// Throw panics, if there is an error. Rob Pike must never know.
func Throw(err error) {
	if err != nil {
		panic(err)
	}
}

// HashBuffer computes a truncated MD5 hash from a buffer
func HashBuffer(buf []byte) string {
	hasher := md5.New()
	_, err := hasher.Write(buf)
	Throw(err)
	return hex.EncodeToString(hasher.Sum(nil))[:16]
}

// MarshalJSON is a helper for marshaling JSON and handling the error
func MarshalJSON(input interface{}) []byte {
	data, err := json.Marshal(input)
	Throw(err)
	return data
}

// UnmarshalJSON is a helper for unmarshalling JSON and handling the error
func UnmarshalJSON(data []byte, store interface{}) {
	Throw(json.Unmarshal(data, store))
}

// CopyFile reads a file from disk and copies it into the writer
func CopyFile(path string, writer io.Writer) {
	file, err := os.Open(path)
	Throw(err)
	defer file.Close()
	_, err = io.Copy(writer, file)
	Throw(err)
}

// IDToString is a  helper for converting a post ID to a string for JSON keys
func IDToString(id uint64) string {
	return strconv.FormatUint(id, 10)
}

// LogError logs an error with its stack trace
func LogError(req *http.Request, err interface{}) {
	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	log.Printf("panic serving %v: %v\n%s", req.RemoteAddr, err, buf)
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
