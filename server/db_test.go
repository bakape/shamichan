package server

import (
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
	"reflect"
)

type DB struct{}

var _ = Suite(&DB{})

// mockDatabase mocks a rethinkDB database for testing purposes. It maps
// generated query strings to their responces
type mockDatabase map[string]interface{}

// mockQuery mocks a query of the rethinkDB database by implementing the
// Database interface
type mockQuery struct {
	query    r.Term
	database mockDatabase
	c        *C
}

func (m mockQuery) Exec() {
	m.verify()
}

func (m mockQuery) One(res interface{}) {
	m.verify()
	m.deference(res)
}

func (m mockQuery) All(res interface{}) {
	m.verify()
	m.deference(res)
}

// verify asserts the generated reQL query string matches the predifined one
func (m mockQuery) verify() {
	q := m.query.String()
	_, ok := m.database[q]
	m.c.Assert(ok, Equals, true, Commentf("invalid query %s", q))
}

// deference copies the stored query result into the target pointer
func (m mockQuery) deference(res interface{}) {
	input := reflect.ValueOf(res)
	if input.Kind() != reflect.Ptr {
		panic("Value must be pointer")
	}
	input.Elem().Set(reflect.ValueOf(m.database[m.query.String()]))
}

// setMockDatabase sets a mockDatabase instance to be used for a test. Should
// be called with a new mockDatabase instance for each test.
func setMockDatabase(m mockDatabase, c *C) {
	db = func() func(r.Term) Database {
		return func(query r.Term) Database {
			return mockQuery{
				query:    query,
				database: m,
				c:        c,
			}
		}
	}
}

func (d *DB) TestParentThread(c *C) {
	const res = uint64(1)
	m := mockDatabase{
		`r.Table("posts").Get(2).Field("op")`: res,
	}
	setMockDatabase(m, c)
	c.Assert(parentThread(2), Equals, res)
}

func (d *DB) TestParentBoard(c *C) {
	m := mockDatabase{
		`r.Table("posts").Get(1).Field("board")`: "a",
	}
	setMockDatabase(m, c)
	c.Assert(parentBoard(1), Equals, "a")
}

func (d *DB) TestValidateOP(c *C) {
	m := mockDatabase{
		`r.Table("posts").Get(1).Field("board")`: "a",
		`r.Table("posts").Get(1).Field("op")`:    uint64(1),
	}
	setMockDatabase(m, c)
	c.Assert(validateOP(1, "a"), Equals, true)

	m = mockDatabase{
		`r.Table("posts").Get(1).Field("board")`: "c",
		`r.Table("posts").Get(1).Field("op")`:    uint64(1),
	}
	setMockDatabase(m, c)
	c.Assert(validateOP(1, "a"), Equals, false)

	m = mockDatabase{
		`r.Table("posts").Get(1).Field("board")`: "a",
		`r.Table("posts").Get(1).Field("op")`:    uint64(2),
	}
	setMockDatabase(m, c)
	c.Assert(validateOP(1, "a"), Equals, false)
}

func (d *DB) TestGetThread(c *C) {
	c.Assert(getThread(1).String(), Equals, `r.Table("threads").Get(1)`)
}

func (d *DB) TestPostCounter(c *C) {
	const res = uint64(1)
	m := mockDatabase{
		`r.Table("main").Get("info").Field("postCtr")`: res,
	}
	setMockDatabase(m, c)
	c.Assert(postCounter(), Equals, res)
}

func (d *DB) TestBoardCounter(c *C) {
	const res = uint64(1)
	m := mockDatabase{
		`r.Table("main").Get("histCounts").Field("a").Default(0)`: res,
	}
	setMockDatabase(m, c)
	c.Assert(boardCounter("a"), Equals, res)
}

func (d *DB) TestThreadCounter(c *C) {
	const res = uint64(4)
	m := mockDatabase{
		`r.Table("posts").GetAll(1, index="op").Count().Sub(1)`: res,
	}
	setMockDatabase(m, c)
	c.Assert(threadCounter(1), Equals, res)
}
