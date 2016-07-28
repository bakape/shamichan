package parser

import (
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	. "gopkg.in/check.v1"
)

func (*Tests) TestParseName(c *C) {
	(*config.Get()).Salt = "123"

	samples := [...]struct {
		in, name, trip string
	}{
		{"", "", ""},
		{"name", "name", ""},
		{"#test", "", ".CzKQna1OU"},
		{"name#test", "name", ".CzKQna1OU"},
		{"##test", "", "mb8h72.d9g"},
		{"name##test", "name", "mb8h72.d9g"},
		{"  name##test ", "name", "mb8h72.d9g"},
	}

	for _, s := range samples {
		name, trip, err := ParseName(s.in)
		c.Assert(err, IsNil)
		c.Assert(name, Equals, s.name)
		c.Assert(trip, Equals, s.trip)
	}
}

func (*Tests) TestParseSubject(c *C) {
	samples := [...]struct {
		in, out string
		err     error
	}{
		{"", "", nil},
		{randomString(maxLengthSubject+1, c), "", ErrTooLong("subject")},
		{" abc ", "abc", nil},
	}

	for _, s := range samples {
		sub, err := ParseSubject(s.in)
		c.Assert(err, Equals, s.err)
		if s.err == nil {
			c.Assert(sub, Equals, s.out)
		}
	}
}

func randomString(length int, c *C) string {
	s, err := auth.RandomID(length)
	c.Assert(err, IsNil)
	return s[:length]
}

func (*Tests) TestVerifyPostPassword(c *C) {
	samples := [...]struct {
		in  string
		err error
	}{
		{"", errNoPostPassword},
		{randomString(maxLengthPostPassword+1, c), ErrTooLong("post password")},
		{randomString(maxLengthPostPassword, c), nil},
	}

	for _, s := range samples {
		c.Assert(VerifyPostPassword(s.in), Equals, s.err)
	}
}
