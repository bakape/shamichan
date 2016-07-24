package util

import (
	"bytes"
	"errors"
	"log"
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Util struct{}

var _ = Suite(&Util{})

func (*Util) TestWrapError(c *C) {
	err := errors.New("foo")
	wrapped := WrapError("bar", err)
	c.Assert(wrapped.Error(), Equals, "bar: foo")
}

func (*Util) TestHashBuffer(c *C) {
	c.Assert(HashBuffer([]byte{1, 2, 3}), Equals, "5289df737df57326")
}

type jsonSample struct {
	A int    `json:"a"`
	B string `json:"b"`
}

func (*Util) TestCopyFile(c *C) {
	buf := new(bytes.Buffer)
	c.Assert(CopyFile("./test/frontpage.html", buf), IsNil)
	c.Assert(buf.String(), Equals, "<!doctype html><html></html>\n")
}

func (*Util) TestIDToString(c *C) {
	c.Assert(IDToString(1), Equals, "1")
}

func (*Util) TestLogError(c *C) {
	err := errors.New("foo")
	log := captureLog(func() {
		LogError("::1", err)
	})
	assertLog(c, strings.Split(log, "\n")[0], "panic serving ::1: foo")
}

func assertLog(c *C, input, standard string) {
	c.Assert(input, Matches, `\d+/\d+/\d+ \d+:\d+:\d+ `+standard)
}

func captureLog(fn func()) string {
	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	fn()
	log.SetOutput(os.Stdout)
	return buf.String()
}

func (*Util) TestRandomID(c *C) {
	hash, err := RandomID(32)
	c.Assert(err, IsNil)
	c.Assert(hash, Matches, "^.{43}$")
}

func (*Util) TestWaterfall(c *C) {
	// All pass
	var wasRun int
	fn := func() error {
		wasRun++
		return nil
	}
	fns := []func() error{fn, fn}
	c.Assert(Waterfall(fns), IsNil)
	c.Assert(wasRun, Equals, 2)

	// 2nd function returns error
	wasRun = 0
	fns = []func() error{
		fn,
		func() error {
			wasRun++
			return errors.New("foo")
		},
		fn,
	}
	c.Assert(Waterfall(fns), ErrorMatches, "foo")
	c.Assert(wasRun, Equals, 2)
}

func (*Util) TestPasswordHash(c *C) {
	const (
		id       = "123"
		password = "123456"
	)
	hash, err := PasswordHash(id, password)
	c.Assert(err, IsNil)

	// Mismatch
	err = ComparePassword(id, password+"1", hash)
	c.Assert(err, Equals, bcrypt.ErrMismatchedHashAndPassword)

	// Correct
	err = ComparePassword(id, password, hash)
	c.Assert(err, IsNil)
}
