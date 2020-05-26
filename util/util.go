// Package util contains various general utility functions used throughout
// the project.
package util

import (
	"crypto/md5"
	"encoding/base64"
)

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
	ch := make(chan error, len(fns)) // Don't leak goroutines on error
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
