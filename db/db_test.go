package db

import (
	"github.com/bakape/meguca/types"
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
	thread, err := parentThread(2)
	c.Assert(err, IsNil)
	c.Assert(thread, Equals, int64(1))

	// Post does not exist
	thread, err = parentThread(15)
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

func (*DBSuite) TestReplicationLog(c *C) {
	std := [][]byte{{1}, {2}, {3}}
	thread := types.DatabaseThread{
		ID:  1,
		Log: std,
	}
	c.Assert(DB(r.Table("threads").Insert(thread)).Exec(), IsNil)
	log, err := ReplicationLog(1)
	c.Assert(err, IsNil)
	c.Assert(log, DeepEquals, std)
}
