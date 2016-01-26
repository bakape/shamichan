package server

import (
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*DB) TestParentThread(c *C) {
	db()(r.Table("posts").Insert(map[string]int{
		"id": 2,
		"op": 1,
	})).Exec()
	c.Assert(parentThread(2), Equals, uint64(1))
}

func (*DB) TestParentBoard(c *C) {
	db()(r.Table("posts").Insert(map[string]interface{}{
		"id":    1,
		"board": "a",
	})).Exec()
	c.Assert(parentBoard(1), Equals, "a")
}

func (*DB) TestValidateOP(c *C) {
	db()(r.Table("posts").Insert(map[string]interface{}{
		"id":    1,
		"op":    1,
		"board": "a",
	})).Exec()
	c.Assert(validateOP(1, "a"), Equals, true)

	db()(r.Table("posts").Get(1).Update(map[string]int{"op": 2})).Exec()
	c.Assert(validateOP(1, "a"), Equals, false)

	db()(r.Table("posts").Get(1).Update(map[string]interface{}{
		"board": "a",
		"op":    2,
	})).Exec()
	c.Assert(validateOP(1, "a"), Equals, false)
}

func (*DB) TestGetThread(c *C) {
	c.Assert(getThread(1).String(), Equals, `r.Table("threads").Get(1)`)
}

func (*DB) TestPostCounter(c *C) {
	db()(r.Table("main").Insert(map[string]interface{}{
		"id":      "info",
		"postCtr": 1,
	})).Exec()
	c.Assert(postCounter(), Equals, uint64(1))
}

func (*DB) TestBoardCounter(c *C) {
	db()(r.Table("main").Insert(map[string]string{"id": "histCounts"})).Exec()
	c.Assert(boardCounter("a"), Equals, uint64(0))

	db()(r.Table("main").Get("histCounts").Update(map[string]int{
		"a": 1,
	})).Exec()
	c.Assert(boardCounter("a"), Equals, uint64(1))
}

func (*DB) TestThreadCounter(c *C) {
	db()(r.Table("posts").Insert(map[string]int{
		"id": 1,
		"op": 1,
	})).Exec()
	c.Assert(threadCounter(1), Equals, uint64(0))

	db()(r.Table("posts").Insert(map[string]int{
		"id": 2,
		"op": 1,
	})).Exec()
	c.Assert(threadCounter(1), Equals, uint64(1))
}

func (*DB) TestDatabaseHelper(c *C) {
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
