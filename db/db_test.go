package db

import (
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*DBSuite) TestParentThread(c *C) {
	std := map[string]int{
		"id": 2,
		"op": 1,
	}
	c.Assert(DB()(r.Table("posts").Insert(std)).Exec(), IsNil)
	thread, err := parentThread(2)
	c.Assert(err, IsNil)
	c.Assert(thread, Equals, uint64(1))

	// Post does not exist
	thread, err = parentThread(15)
	c.Assert(err, IsNil)
	c.Assert(thread, Equals, uint64(0))
}

func (*DBSuite) TestParentBoard(c *C) {
	std := map[string]interface{}{
		"id":    1,
		"board": "a",
	}
	c.Assert(DB()(r.Table("posts").Insert(std)).Exec(), IsNil)

	b, err := parentBoard(1)
	c.Assert(err, IsNil)
	c.Assert(b, Equals, "a")

	// Post does not exist
	b, err = parentBoard(15)
	c.Assert(err, IsNil)
	c.Assert(b, Equals, "")
}

func (*DBSuite) TestValidateOP(c *C) {
	std := map[string]interface{}{
		"id":    1,
		"board": "a",
	}
	c.Assert(DB()(r.Table("threads").Insert(std)).Exec(), IsNil)

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
	std := map[string]interface{}{
		"id":      "info",
		"postCtr": 1,
	}
	c.Assert(DB()(r.Table("main").Insert(std)).Exec(), IsNil)

	count, err := PostCounter()
	c.Assert(err, IsNil)
	c.Assert(count, Equals, uint64(1))
}

func (*DBSuite) TestBoardCounter(c *C) {
	std := map[string]string{"id": "histCounts"}
	c.Assert(DB()(r.Table("main").Insert(std)).Exec(), IsNil)

	count, err := BoardCounter("a")
	c.Assert(err, IsNil)
	c.Assert(count, Equals, uint64(0))

	update := map[string]int{"a": 1}
	err = DB()(r.Table("main").Get("histCounts").Update(update)).Exec()
	c.Assert(err, IsNil)

	count, err = BoardCounter("a")
	c.Assert(err, IsNil)
	c.Assert(count, Equals, uint64(1))
}

func (*DBSuite) TestThreadCounter(c *C) {
	std := map[string]int{
		"id": 1,
		"op": 1,
	}
	c.Assert(DB()(r.Table("posts").Insert(std)).Exec(), IsNil)

	count, err := ThreadCounter(1)
	c.Assert(err, IsNil)
	c.Assert(count, Equals, uint64(0))

	more := map[string]int{
		"id": 2,
		"op": 1,
	}
	c.Assert(DB()(r.Table("posts").Insert(more)).Exec(), IsNil)

	count, err = ThreadCounter(1)
	c.Assert(err, IsNil)
	c.Assert(count, Equals, uint64(1))
}

func (*DBSuite) TestDatabaseHelper(c *C) {
	standard := Document{"doc"}
	helper := DatabaseHelper{r.Table("main").Insert(standard)}
	err := helper.Exec()
	c.Assert(err, IsNil)

	var doc Document
	helper = DatabaseHelper{r.Table("main").Get("doc")}
	err = helper.One(&doc)
	c.Assert(err, IsNil)
	c.Assert(doc, DeepEquals, standard)

	var docs []Document
	helper = DatabaseHelper{r.Table("main")}
	err = helper.All(&docs)
	c.Assert(err, IsNil)
	c.Assert(docs, DeepEquals, []Document{standard})
}
