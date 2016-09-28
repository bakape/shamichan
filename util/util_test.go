package util

import (
	"errors"
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
	c.Assert(HashBuffer([]byte{1, 2, 3}), Equals, "Uonfc331cyb83SJZevsfrA")
}

type jsonSample struct {
	A int    `json:"a"`
	B string `json:"b"`
}

func (*Util) TestIDToString(c *C) {
	c.Assert(IDToString(1), Equals, "1")
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
