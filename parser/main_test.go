package parser

import (
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Tests struct{}

var _ = Suite(&Tests{})

func (*Tests) SetUpSuite(c *C) {
	db.DBName = db.UniqueDBName()
	c.Assert(db.Connect(), IsNil)
	c.Assert(db.InitDB(), IsNil)
}

func (*Tests) TearDownSuite(c *C) {
	c.Assert(r.DBDrop(db.DBName).Exec(db.RSession), IsNil)
	c.Assert(db.RSession.Close(), IsNil)
}

func (*Tests) SetUpTest(c *C) {
	config.Set(config.Configs{})
	c.Assert(db.ClearTables(), IsNil)
}

func (*Tests) TestStripPsuedoWhitespace(c *C) {
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
