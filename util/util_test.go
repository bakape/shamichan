package util

import (
	"bytes"
	"errors"
	"log"
	"os"
	"strings"
	"testing"

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
