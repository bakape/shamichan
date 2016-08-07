package websockets

import (
	"bytes"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/parser"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

var (
	dummyLog = [][]byte{
		{1, 2, 3},
		{3, 4, 5},
	}

	sampleThread = types.DatabaseThread{
		ID:  1,
		Log: dummyLog,
		Posts: map[int64]types.DatabasePost{
			2: {
				Post: types.Post{
					Editing: true,
					ID:      2,
					Body:    "abc",
				},
			},
		},
	}
)

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
	stdMsg, err := encodeMessage(messageBacklink, types.LinkMap{
		10: {
			OP:    9,
			Board: "a",
		},
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
	sv := newWSServer(c)
	defer sv.Close()

	fns := [...]func([]byte, *Client) error{appendRune, backspace, closePost}
	for _, fn := range fns {
		cl, _ := sv.NewClient()
		c.Assert(fn(nil, cl), Equals, errNoPostOpen)
	}
}

func (*DB) TestLineEmpty(c *C) {
	fns := [...]func([]byte, *Client) error{backspace}
	sv := newWSServer(c)
	defer sv.Close()

	for _, fn := range fns {
		cl, _ := sv.NewClient()
		cl.openPost.id = 1
		c.Assert(fn(nil, cl), Equals, errLineEmpty)
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
	thread := sampleThread
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}

	c.Assert(appendRune([]byte("100"), cl), IsNil)

	c.Assert(cl.openPost.bodyLength, Equals, 4)
	c.Assert(cl.openPost.String(), Equals, "abcd")
	assertBody(2, "abcd", c)

	assertRepLog(2, append(dummyLog, []byte(`03[1,100]`)), c)
}

func assertBody(id int64, body string, c *C) {
	var res string
	q := db.FindPost(2).Field("body")
	c.Assert(db.One(q, &res), IsNil)
	c.Assert(res, Equals, body)
}

func assertRepLog(id int64, log [][]byte, c *C) {
	var res [][]byte
	q := db.FindParentThread(id).Field("log")
	c.Assert(db.All(q, &res), IsNil)
	c.Assert(log, DeepEquals, log)
}

func (*DB) TestAppendNewline(c *C) {
	thread := sampleThread
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		board:      "a",
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}

	conf := config.BoardConfigs{
		ID: "a",
	}
	c.Assert(db.Write(r.Table("boards").Insert(conf)), IsNil)

	c.Assert(appendRune([]byte("10"), cl), IsNil)

	c.Assert(cl.openPost.bodyLength, Equals, 4)
	c.Assert(cl.openPost.String(), Equals, "")
	assertBody(2, "abc\n", c)
	assertRepLog(2, append(dummyLog, []byte("03[1,10]")), c)
}

func (*DB) TestAppendNewlineWithHashCommand(c *C) {
	thread := types.DatabaseThread{
		ID:  1,
		Log: dummyLog,
		Posts: map[int64]types.DatabasePost{
			2: {
				Post: types.Post{
					ID:   2,
					Body: "#flip",
				},
			},
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		board:      "a",
		Buffer:     *bytes.NewBuffer([]byte("#flip")),
	}

	conf := config.BoardConfigs{
		ID: "a",
		PostParseConfigs: config.PostParseConfigs{
			HashCommands: true,
		},
	}
	c.Assert(db.Write(r.Table("boards").Insert(conf)), IsNil)

	c.Assert(appendRune([]byte("10"), cl), IsNil)

	var typ int
	q := db.FindPost(2).Field("commands").AtIndex(0).Field("type")
	c.Assert(db.One(q, &typ), IsNil)
	c.Assert(typ, Equals, int(types.Flip))

	var log []byte
	q = db.FindParentThread(2).Field("log").Nth(-1)
	c.Assert(db.One(q, &log), IsNil)
	c.Assert(string(log), Matches, `09\{"type":1,"val":(?:true|false)\}`)
}

func (*DB) TestAppendNewlineWithLinks(c *C) {
	threads := []types.DatabaseThread{
		{
			ID:    1,
			Board: "a",
			Log:   [][]byte{},
			Posts: map[int64]types.DatabasePost{
				2: {
					Post: types.Post{
						ID:   2,
						Body: " >>22 ",
					},
				},
			},
		},
		{
			ID:    21,
			Board: "c",
			Log:   [][]byte{},
			Posts: map[int64]types.DatabasePost{
				22: {
					Post: types.Post{
						ID: 22,
					},
				},
			},
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(threads)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		board:      "a",
		Buffer:     *bytes.NewBuffer([]byte(" >>22 ")),
	}

	conf := config.BoardConfigs{
		ID: "a",
	}
	c.Assert(db.Write(r.Table("boards").Insert(conf)), IsNil)

	c.Assert(appendRune([]byte("10"), cl), IsNil)

	std := [...]struct {
		id    int64
		log   string
		field string
		val   types.LinkMap
	}{
		{
			2,
			`07{"22":{"op":21,"board":"c"}}`,
			"links",
			types.LinkMap{
				22: {
					OP:    21,
					Board: "c",
				},
			},
		},
		{
			22,
			`08{"2":{"op":1,"board":"a"}}`,
			"backlinks",
			types.LinkMap{
				2: {
					OP:    1,
					Board: "a",
				},
			},
		},
	}
	for _, s := range std {
		assertRepLog(s.id, [][]byte{[]byte(s.log)}, c)

		var links types.LinkMap
		q := db.FindPost(s.id).Field(s.field)
		c.Assert(db.One(q, &links), IsNil)
		c.Assert(links, DeepEquals, s.val)
	}
}

func (*DB) TestBackspace(c *C) {
	thread := sampleThread
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}

	c.Assert(backspace([]byte{}, cl), IsNil)

	c.Assert(cl.openPost.String(), Equals, "ab")
	c.Assert(cl.openPost.bodyLength, Equals, 2)

	assertRepLog(2, append(dummyLog, []byte("041")), c)
	assertBody(2, "ab", c)
}

func (*DB) TestClosePost(c *C) {
	thread := sampleThread
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	conf := config.BoardConfigs{
		ID: "a",
	}
	c.Assert(db.Write(r.Table("boards").Insert(conf)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		board:      "a",
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}

	c.Assert(closePost([]byte{}, cl), IsNil)

	c.Assert(cl.openPost, DeepEquals, openPost{})
	assertRepLog(2, append(dummyLog, []byte("062")), c)
	assertBody(2, "abc", c)

	var editing bool
	q := db.FindPost(2).Field("editing")
	c.Assert(db.One(q, &editing), IsNil)
	c.Assert(editing, Equals, false)
}
