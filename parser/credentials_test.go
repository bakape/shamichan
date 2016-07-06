package parser

import (
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
