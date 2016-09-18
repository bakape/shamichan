package parser

import (
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*Tests) TestNoLinks(c *C) {
	links, err := parseLinks([]byte("foo bar baz"))
	c.Assert(err, IsNil)
	c.Assert(links, IsNil)
}

func (*Tests) TestLinks(c *C) {
	thread := types.DatabaseThread{
		ID:    2,
		Board: "a",
		Posts: map[int64]types.DatabasePost{
			4: types.DatabasePost{},
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	links, err := parseLinks([]byte(">>>1 >>4 "))
	c.Assert(err, IsNil)
	c.Assert(links, DeepEquals, types.LinkMap{
		4: types.Link{
			OP:    2,
			Board: "a",
		},
	})
}

func (*Tests) TestAllLinksInvalid(c *C) {
	links, err := parseLinks([]byte(" >>1 >>2 >>33"))
	c.Assert(err, IsNil)
	c.Assert(links, IsNil)
}
