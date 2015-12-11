// Package util contains various general helper functions
package util

// Throw panics, if there is an error. Rob Pike must never know.
func Throw(err error) {
	if err != nil {
		panic(err)
	}
}
