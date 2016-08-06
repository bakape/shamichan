package websockets

import (
	"bytes"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/parser"
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

func (*DB) TestNoOpenPost(c *C) {
	fns := [...]func([]byte, *Client) error{appendRune}
	sv := newWSServer(c)
	defer sv.Close()

	for _, fn := range fns {
		cl, _ := sv.NewClient()
		c.Assert(fn(nil, cl), Equals, errNoPostOpen)
	}
}

func (*DB) TestAppendBodyTooLong(c *C) {
	sv := newWSServer(c)
	defer sv.Close()

	cl, _ := sv.NewClient()
	cl.openPost.id = 1
	cl.openPost.bodyLength = parser.MaxLengthBody

	c.Assert(appendRune(nil, cl), Equals, parser.ErrBodyTooLong)
}

func (*DB) TestAppendRune(c *C) {
	thread := types.DatabaseThread{
		ID:  1,
		Log: dummyLog,
		Posts: map[int64]types.DatabasePost{
			1: {
				Post: types.Post{
					ID:   1,
					Body: "abc",
				},
			},
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         1,
		op:         1,
		bodyLength: 3,
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}

	c.Assert(appendRune([]byte("100"), cl), IsNil)

	c.Assert(cl.openPost.bodyLength, Equals, 4)
	c.Assert(cl.openPost.String(), Equals, "abcd")

	var body string
	q := db.FindPost(1).Field("body")
	c.Assert(db.One(q, &body), IsNil)
	c.Assert(body, Equals, "abcd")

	stdMsg := []byte(`03[1,100]`)
	var log [][]byte
	q = db.FindParentThread(1).Field("log")
	c.Assert(db.All(q, &log), IsNil)
	c.Assert(log, DeepEquals, append(dummyLog, stdMsg))
}
