package util

import (
	"bytes"
	"errors"
	. "gopkg.in/check.v1"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
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
	hash, err := HashBuffer([]byte{1, 2, 3})
	c.Assert(err, IsNil)
	c.Assert(hash, Equals, "5289df737df57326")
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
	req, e := http.NewRequest("GET", "/", nil)
	c.Assert(e, IsNil)
	err := errors.New("foo")
	req.RemoteAddr = "::1"
	log := captureLog(func() {
		LogError(req, err)
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
	c.Assert(RandomID(32), Matches, "^[0-9a-zA-Z]{32}$")
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
