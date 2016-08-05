package websockets

import (
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

var dummyLog = [][]byte{
	{1, 2, 3},
	{3, 4, 5},
}

func (*DB) TestWriteBacklinks(c *C) {
	threads := []types.DatabaseThread{
		{
			ID: 1,
			Posts: map[int64]types.DatabasePost{
				1: {
					Post: types.Post{
						ID: 1,
					},
				},
				2: {
					Post: types.Post{
						ID: 2,
					},
				},
			},
			Log: dummyLog,
		},
		{
			ID: 5,
			Posts: map[int64]types.DatabasePost{
				7: {
					Post: types.Post{
						ID: 7,
					},
				},
			},
			Log: dummyLog,
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(threads)), IsNil)

	for _, dest := range [...]int64{1, 2, 7, 8} {
		c.Assert(writeBacklink(10, 9, "a", dest), IsNil)
	}

	// Assert each existong post had a backlink inserted
	std := types.Link{
		OP:    9,
		Board: "a",
	}
	stdMsg, err := encodeMessage(messageBacklink, backlinkInsertionMessage{
		ID:    10,
		OP:    9,
		Board: "a",
	})
	c.Assert(err, IsNil)

	for _, id := range [...]int64{1, 2, 7} {
		var link types.Link
		q := db.FindPost(id).Field("backlinks").Field("10")
		c.Assert(db.One(q, &link), IsNil)
		c.Assert(link, Equals, std)

		var constains bool
		q = db.FindParentThread(id).Field("log").Contains(stdMsg)
		c.Assert(db.One(q, &constains), IsNil)
		c.Assert(constains, Equals, true)
	}
}
