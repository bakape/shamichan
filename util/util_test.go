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
	wrapped := WrapError{"bar", err}
	c.Assert(wrapped.Error(), Equals, "bar: foo")
}

func (*Util) TestThrowNoError(c *C) {
	defer c.Assert(recover(), IsNil)
	Throw(nil)
}

func (*Util) TestThrowWithError(c *C) {
	err := errors.New("foo")
	defer func() {
		c.Assert(recover(), DeepEquals, err)
	}()
	Throw(err)
}

func (*Util) TestHashBuffer(c *C) {
	c.Assert(HashBuffer([]byte{1, 2, 3}), Equals, "5289df737df57326")
}

type jsonSample struct {
	A int    `json:"a"`
	B string `json:"b"`
}

func (*Util) TestMarshalJSON(c *C) {
	s := jsonSample{1, "b"}
	c.Assert(string(MarshalJSON(s)), Equals, `{"a":1,"b":"b"}`)
}

func (*Util) TestUnmarshalJSON(c *C) {
	const json = `{"a":1,"b":"b"}`
	var store jsonSample
	result := jsonSample{1, "b"}
	UnmarshalJSON([]byte(json), &store)
	c.Assert(store, DeepEquals, result)
}

func (*Util) TestCopyFile(c *C) {
	buf := new(bytes.Buffer)
	CopyFile("./test/frontpage.html", buf)
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
