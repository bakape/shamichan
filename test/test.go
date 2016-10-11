// Package test contains utility functions used throughout the project in tests
package test

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"testing"
)

// LogUnexpected fails the test and prints the values in an
// `expected: X got: Y` format
func LogUnexpected(t *testing.T, expected, got interface{}) {
	t.Fatalf("\nexpected: %#v\ngot:      %#v", expected, got)
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
	t.Fatalf("unexpected error: %#v", err)
}

// AssertFileEquals reads a file from disk and asserts it eaquals the standard
// buffer
func AssertFileEquals(t *testing.T, path string, std []byte) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf, std) {
		t.Fatalf("files not equal: `%s` : `%s`", string(std), string(buf))
	}
}
