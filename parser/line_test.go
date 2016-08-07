package parser

import (
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*Tests) TestParseHashCommandLine(c *C) {
	conf := config.BoardConfigs{
		ID: "a",
		PostParseConfigs: config.PostParseConfigs{
			HashCommands: true,
		},
	}
	c.Assert(db.Write(r.Table("boards").Insert(conf)), IsNil)

	links, com, err := ParseLine([]byte("#flip"), "a")
	c.Assert(err, IsNil)
	c.Assert(links, IsNil)
	c.Assert(com.Type, Equals, types.Flip)
}

func (*Tests) TestHashCommandsDisabled(c *C) {
	conf := config.BoardConfigs{
		ID: "a",
	}
	c.Assert(db.Write(r.Table("boards").Insert(conf)), IsNil)

	links, com, err := ParseLine([]byte("#flip"), "a")
	c.Assert(err, IsNil)
	c.Assert(links, IsNil)
	c.Assert(com.Val, IsNil)
}
