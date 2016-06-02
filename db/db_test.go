package db

import (
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*DBSuite) TestParentThread(c *C) {
	std := types.DatabaseThread{
		ID:    1,
		Board: "a",
		Posts: map[string]types.Post{
			"2": {
				ID: 2,
			},
		},
	}
	c.Assert(DB(r.Table("threads").Insert(std)).Exec(), IsNil)
	thread, err := ParentThread(2)
	c.Assert(err, IsNil)
	c.Assert(thread, Equals, int64(1))

	// Post does not exist
	thread, err = ParentThread(15)
	c.Assert(err, IsNil)
	c.Assert(thread, Equals, int64(0))
}

func (*DBSuite) TestValidateOP(c *C) {
	std := types.DatabaseThread{
		ID:    1,
		Board: "a",
	}
	c.Assert(DB(r.Table("threads").Insert(std)).Exec(), IsNil)

	v, err := ValidateOP(1, "a")
	c.Assert(err, IsNil)
	c.Assert(v, Equals, true)

	// Thread does not exist
	v, err = ValidateOP(15, "a")
	c.Assert(err, IsNil)
	c.Assert(v, Equals, false)
}

func (*DBSuite) TestGetThread(c *C) {
	c.Assert(getThread(1).String(), Equals, `r.Table("threads").Get(1)`)
}

func (*DBSuite) TestPostCounter(c *C) {
	std := infoDocument{
		Document: Document{"info"},
		PostCtr:  1,
	}
	c.Assert(DB(r.Table("main").Insert(std)).Exec(), IsNil)

	count, err := PostCounter()
	c.Assert(err, IsNil)
	c.Assert(count, Equals, int64(1))
}

func (*DBSuite) TestBoardCounter(c *C) {
	std := Document{"histCounts"}
	c.Assert(DB(r.Table("main").Insert(std)).Exec(), IsNil)

	count, err := BoardCounter("a")
	c.Assert(err, IsNil)
	c.Assert(count, Equals, int64(0))

	update := map[string]int{"a": 1}
	c.Assert(DB(GetMain("histCounts").Update(update)).Exec(), IsNil)

	count, err = BoardCounter("a")
	c.Assert(err, IsNil)
	c.Assert(count, Equals, int64(1))
}

func (*DBSuite) TestThreadCounter(c *C) {
	std := types.DatabaseThread{
		ID: 1,
		Log: [][]byte{
			{1},
			{2},
			{3},
		},
	}
	c.Assert(DB(r.Table("threads").Insert(std)).Exec(), IsNil)

	count, err := ThreadCounter(1)
	c.Assert(err, IsNil)
	c.Assert(count, Equals, int64(3))
}

func (*DBSuite) TestDatabaseHelper(c *C) {
	standard := Document{"doc"}
	helper := DatabaseHelper{r.Table("main").Insert(standard)}
	err := helper.Exec()
	c.Assert(err, IsNil)

	var doc Document
	helper = DatabaseHelper{GetMain("doc")}
	err = helper.One(&doc)
	c.Assert(err, IsNil)
	c.Assert(doc, DeepEquals, standard)

	var docs []Document
	helper = DatabaseHelper{r.Table("main")}
	err = helper.All(&docs)
	c.Assert(err, IsNil)
	c.Assert(docs, DeepEquals, []Document{standard})
}

func (*DBSuite) TestStreamUpdates(c *C) {
	thread := types.DatabaseThread{ID: 1}
	c.Assert(DB(r.Table("threads").Insert(thread)).Exec(), IsNil)

	// Empty log
	read := make(chan []byte, 1)
	closer := new(util.AtomicCloser)
	initial, err := StreamUpdates(1, read, closer)
	c.Assert(err, IsNil)
	c.Assert(initial, DeepEquals, [][]byte{})

	addition := []byte{1, 0, 0, 3, 2}
	log := [][]byte{addition}
	update := map[string][][]byte{"log": log}
	c.Assert(DB(getThread(1).Update(update)).Exec(), IsNil)
	c.Assert(<-read, DeepEquals, addition)
	closer.Close()

	// Existing data
	read = make(chan []byte, 1)
	closer = new(util.AtomicCloser)
	initial, err = StreamUpdates(1, read, closer)
	c.Assert(err, IsNil)
	c.Assert(initial, DeepEquals, log)
	closer.Close()
}

func (*DBSuite) TestFindNonexistantImageThumb(c *C) {
	img, err := FindImageThumb("sha")
	c.Assert(err, IsNil)
	c.Assert(img, DeepEquals, types.ProtoImage{})
}

func (*DBSuite) TestFindImageThumb(c *C) {
	thumbnailed := types.ProtoImage{
		File:     "123",
		SHA1:     "foo",
		FileType: 1,
		Posts:    1,
	}
	insertProtoImage(thumbnailed, c)

	img, err := FindImageThumb("foo")
	c.Assert(err, IsNil)
	thumbnailed.Posts++
	c.Assert(img, DeepEquals, thumbnailed)

	assertImageRefCount("123", 2, c)
}

func insertProtoImage(img types.ProtoImage, c *C) {
	c.Assert(DB(r.Table("images").Insert(img)).Exec(), IsNil)
}

func assertImageRefCount(id string, count int, c *C) {
	var posts int
	c.Assert(DB(GetImage(id).Field("posts")).One(&posts), IsNil)
	c.Assert(posts, Equals, count)
}

func (*DBSuite) TestDecreaseImageRefCount(c *C) {
	const id = "123"
	img := types.ProtoImage{
		File:  id,
		Posts: 2,
	}
	insertProtoImage(img, c)

	c.Assert(UnreferenceImage(id), IsNil)
	assertImageRefCount(id, 1, c)
}

func (*DBSuite) TestRemoveUnreffedImage(c *C) {
	const id = "123"
	img := types.ProtoImage{
		File:  id,
		Posts: 1,
	}
	insertProtoImage(img, c)

	c.Assert(UnreferenceImage(id), IsNil)

	var noImage bool
	c.Assert(DB(GetImage(id).Eq(nil)).One(&noImage), IsNil)
	c.Assert(noImage, Equals, true)
}
