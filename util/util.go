// Package util contains various general utility functions used throughout
// the project.
package util

import (
	"crypto/md5"
	"encoding/base64"
	"net/http"
	"strings"
	"unicode/utf8"
)

// WrapError wraps error types to create compound error chains
func WrapError(text string, err error) error {
	return WrappedError{
		Text:  text,
		Inner: err,
	}
}

// WrappedError wraps error types to create compound error chains
type WrappedError struct {
	Text  string
	Inner error
}

func (e WrappedError) Error() string {
	text := e.Text
	if e.Inner != nil {
		text += ": " + e.Inner.Error()
	}
	return text
}

// Waterfall executes a slice of functions until the first error returned. This
// error, if any, is returned to the caller.
func Waterfall(fns ...func() error) (err error) {
	for _, fn := range fns {
		err = fn()
		if err != nil {
			break
		}
	}
	return
}

// Parallel executes functions in parallel. The first error is returned, if any.
func Parallel(fns ...func() error) error {
	ch := make(chan error)
	for i := range fns {
		fn := fns[i]
		go func() {
			ch <- fn()
		}()
	}

	for range fns {
		if err := <-ch; err != nil {
			return err
		}
	}

	return nil
}

// HashBuffer computes a base64 MD5 hash from a buffer
func HashBuffer(buf []byte) string {
	hash := md5.Sum(buf)
	return base64.RawStdEncoding.EncodeToString(hash[:])
}

// ConcatStrings efficiently concatenates strings with only one extra allocation
func ConcatStrings(s ...string) string {
	l := 0
	for _, s := range s {
		l += len(s)
	}
	b := make([]byte, 0, l)
	for _, s := range s {
		b = append(b, s...)
	}
	return string(b)
}

// CloneBytes creates a copy of b
func CloneBytes(b []byte) []byte {
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp
}

// SplitPunctuation splits off one byte of leading and trailing punctuation,
// if any, and returns the 3 split parts. If there is no edge punctuation, the
// respective byte = 0.
func SplitPunctuation(word []byte) (leading byte, mid []byte, trailing byte) {
	mid = word

	// Split leading
	if len(mid) < 2 {
		return
	}
	if isPunctuation(mid[0]) {
		leading = mid[0]
		mid = mid[1:]
	}

	// Split trailing
	l := len(mid)
	if l < 2 {
		return
	}
	if isPunctuation(mid[l-1]) {
		trailing = mid[l-1]
		mid = mid[:l-1]
	}

	return
}

// isPunctuation returns, if b is a punctuation symbol
func isPunctuation(b byte) bool {
	switch b {
	case '!', '"', '\'', '(', ')', ',', '-', '.', ':', ';', '?', '[', ']':
		return true
	default:
		return false
	}
}

// SplitPunctuationString splits off one byte of leading and trailing
// punctuation, if any, and returns the 3 split parts. If there is no edge
// punctuation, the respective byte = 0.
func SplitPunctuationString(word string) (
	leading byte, mid string, trailing byte,
) {
	// Generic copy paste :^)

	mid = word

	// Split leading
	if len(mid) < 2 {
		return
	}
	if isPunctuation(mid[0]) {
		leading = mid[0]
		mid = mid[1:]
	}

	// Split trailing
	l := len(mid)
	if l < 2 {
		return
	}
	if isPunctuation(mid[l-1]) {
		trailing = mid[l-1]
		mid = mid[:l-1]
	}

	return
}

// Trim string, while making sure it's still valid unicode, in case a rune was
// split in half
func TrimString(s *string, maxLen int) {
	if len(*s) > maxLen {
		*s = (*s)[:maxLen]
		if !utf8.ValidString(*s) {
			*s = strings.ToValidUTF8(*s, "?")
		}
	}
}

// Adds security options to cookie and sets it in responsewriter
func SetCookie(w http.ResponseWriter, r *http.Request, c *http.Cookie) error {
	// Allow localhost to set cookies on http
	for _, s := range [...]string{"127.0.0.1", "[::1]", "localhost"} {
		// Compare as a prefix to avoid messing with :portnumber
		if strings.HasPrefix(r.Host, s) {
			c.Secure = false
			c.SameSite = http.SameSiteDefaultMode
			http.SetCookie(w, c)

			return nil
		}
	}
	c.Secure = true
	c.SameSite = http.SameSiteNoneMode
	http.SetCookie(w, c)

	return nil
}
