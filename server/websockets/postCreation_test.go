package websockets

import (
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*ClientSuite) TestStripPsuedoWhitespace(c *C) {
	samples := []struct {
		in, out string
	}{
		{"normal", "normal"},
		{"h\u2000e\u200fl\u202al\u202fo\u205f\u206f", "hello"},
		{"日本\u2062語", "日本語"},
	}
	for _, s := range samples {
		c.Assert(stripPsuedoWhitespace(s.in), Equals, s.out)
	}
}

func (DB) TestForcedAnon(c *C) {
	q := r.Table("boards").Insert(config.BoardConfigs{
		ID:         "a",
		ForcedAnon: true,
	})
	c.Assert(db.Write(q), IsNil)

	name, trip, err := parseName("name#trip", "a")
	c.Assert(err, IsNil)
	c.Assert(name, Equals, "")
	c.Assert(trip, Equals, "")
}

func (*DB) TestParseName(c *C) {
	(*config.Get()).Salt = "123"

	samples := []struct {
		in, name, trip string
	}{
		{"", "", ""},
		{"name", "name", ""},
		{"#test", "", ".CzKQna1OU"},
		{"name#test", "name", ".CzKQna1OU"},
		{"##test", "", "mb8h72.d9g"},
		{"name##test", "name", "mb8h72.d9g"},
	}

	for _, s := range samples {
		name, trip, err := parseName(s.in, "a")
		c.Assert(err, IsNil)
		c.Assert(name, Equals, s.name)
		c.Assert(trip, Equals, s.trip)
	}
}
