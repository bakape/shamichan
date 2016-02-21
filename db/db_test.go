package db

import (
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*DBSuite) TestParentThread(c *C) {
	DB()(r.Table("posts").Insert(map[string]int{
		"id": 2,
		"op": 1,
	})).Exec()
	c.Assert(parentThread(2), Equals, uint64(1))

	// Post does not exist
	c.Assert(parentThread(15), Equals, uint64(0))
}

func (*DBSuite) TestParentBoard(c *C) {
	DB()(r.Table("posts").Insert(map[string]interface{}{
		"id":    1,
		"board": "a",
	})).Exec()
	c.Assert(parentBoard(1), Equals, "a")

	// Post does not exist
	c.Assert(parentBoard(15), Equals, "")
}

func (*DBSuite) TestValidateOP(c *C) {
	DB()(r.Table("threads").Insert(map[string]interface{}{
		"id":    1,
		"board": "a",
	})).Exec()
	c.Assert(ValidateOP(1, "a"), Equals, true)

	// Thread does not exist
	c.Assert(ValidateOP(15, "a"), Equals, false)
}

func (*DBSuite) TestGetThread(c *C) {
	c.Assert(getThread(1).String(), Equals, `r.Table("threads").Get(1)`)
}

func (*DBSuite) TestPostCounter(c *C) {
	DB()(r.Table("main").Insert(map[string]interface{}{
		"id":      "info",
		"postCtr": 1,
	})).Exec()
	c.Assert(PostCounter(), Equals, uint64(1))
}

func (*DBSuite) TestBoardCounter(c *C) {
	DB()(r.Table("main").Insert(map[string]string{"id": "histCounts"})).Exec()
	c.Assert(BoardCounter("a"), Equals, uint64(0))

	DB()(r.Table("main").Get("histCounts").Update(map[string]int{
		"a": 1,
	})).Exec()
	c.Assert(BoardCounter("a"), Equals, uint64(1))
}

func (*DBSuite) TestThreadCounter(c *C) {
	DB()(r.Table("posts").Insert(map[string]int{
		"id": 1,
		"op": 1,
	})).Exec()
	c.Assert(ThreadCounter(1), Equals, uint64(0))

	DB()(r.Table("posts").Insert(map[string]int{
		"id": 2,
		"op": 1,
	})).Exec()
	c.Assert(ThreadCounter(1), Equals, uint64(1))
}

func (*DBSuite) TestDatabaseHelper(c *C) {
	standard := Document{"doc"}
	helper := DatabaseHelper{r.Table("main").Insert(standard)}
	helper.Exec()

	var doc Document
	helper = DatabaseHelper{r.Table("main").Get("doc")}
	helper.One(&doc)
	c.Assert(doc, DeepEquals, standard)

	var docs []Document
	helper = DatabaseHelper{r.Table("main")}
	helper.All(&docs)
	c.Assert(docs, DeepEquals, []Document{standard})
}
