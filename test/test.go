// Package test contains utility functions used throughout the project in tests
package test

import (
	"reflect"
	"testing"
)

// LogUnexpected fails the test and prints the values in an
// `expected: X got: Y` format
func LogUnexpected(t *testing.T, expected, got interface{}) {
	t.Fatal("\nexpected: %#v\ngot:      %#v", expected, got)
}

// AssertDeepEquals aserts two values are deeply equal or fails the test, if
// not
func AssertDeepEquals(t *testing.T, res, std interface{}) {
	if !reflect.DeepEqual(res, std) {
		LogUnexpected(t, std, res)
	}
}

// UnexpectedError fails the test with an unexecpted error message
func UnexpectedError(t *testing.T, err error) {
	t.Fatal("unexpected error: %#v", err)
}
