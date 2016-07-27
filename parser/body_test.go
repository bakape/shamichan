package parser

import (
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*Tests) TestBodyAllWhitespace(c *C) {
	samples := [...]string{
		" \n\t  ",
		"\u2000\n\u200f \n",
	}
	for _, s := range samples {
		_, err := BodyParser{}.ParseBody(s)
		c.Assert(err, Equals, ErrOnlyWhitespace)
	}
}

func (Tests) TestBodyTooLong(c *C) {
	body, err := util.RandomID(maxLengthBody + 1)
	c.Assert(err, IsNil)
	_, err = BodyParser{}.ParseBody(body)
	c.Assert(err, Equals, errBodyTooLong)
}

func (*Tests) TestParseBody(c *C) {
	const in = " \u2000\u200ffoo >>7\n#flip\n #flip\n >>7 >>8"
	thread := types.DatabaseThread{
		ID: 1,
		Posts: map[int64]types.Post{
			7: types.Post{},
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	std := BodyParseResults{
		Body: "foo >>7\n#flip\n #flip\n >>7 >>8",
		Links: map[int64]types.Link{
			7: {},
		},
		Commands: []types.Command{
			{
				Type: types.Flip,
				Val:  true,
			},
		},
	}
	bp := BodyParser{
		Config: config.PostParseConfigs{
			HashCommands: true,
		},
	}
	res, err := bp.ParseBody(in)
	c.Assert(err, IsNil)

	// Normalize random aspect of flip
	res.Commands[0].Val = true
	c.Assert(res, DeepEquals, std)
}
