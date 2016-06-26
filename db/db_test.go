package db

import (
	"github.com/bakape/meguca/auth"
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
	c.Assert(Write(r.Table("threads").Insert(std)), IsNil)
	thread, err := ParentThread(2)
	c.Assert(err, IsNil)
	c.Assert(thread, Equals, int64(1))

	// Post does not exist
	thread, err = ParentThread(15)
	c.Assert(err, Equals, r.ErrEmptyResult)
	c.Assert(thread, Equals, int64(0))
}

func (*DBSuite) TestValidateOP(c *C) {
	std := types.DatabaseThread{
		ID:    1,
		Board: "a",
	}
	c.Assert(Write(r.Table("threads").Insert(std)), IsNil)

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
	c.Assert(Write(r.Table("main").Insert(std)), IsNil)

	count, err := PostCounter()
	c.Assert(err, IsNil)
	c.Assert(count, Equals, int64(1))
}

func (*DBSuite) TestBoardCounter(c *C) {
	std := Document{"histCounts"}
	c.Assert(Write(r.Table("main").Insert(std)), IsNil)

	count, err := BoardCounter("a")
	c.Assert(err, IsNil)
	c.Assert(count, Equals, int64(0))

	update := map[string]int{"a": 1}
	c.Assert(Write(GetMain("histCounts").Update(update)), IsNil)

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
	c.Assert(Write(r.Table("threads").Insert(std)), IsNil)

	count, err := ThreadCounter(1)
	c.Assert(err, IsNil)
	c.Assert(count, Equals, int64(3))
}

func (*DBSuite) TestStreamUpdates(c *C) {
	thread := types.DatabaseThread{ID: 1}
	c.Assert(Write(r.Table("threads").Insert(thread)), IsNil)

	// Empty log
	read := make(chan []byte, 1)
	closer := new(util.AtomicCloser)
	initial, err := StreamUpdates(1, read, closer)
	c.Assert(err, IsNil)
	c.Assert(initial, DeepEquals, [][]byte{})

	addition := []byte{1, 0, 0, 3, 2}
	log := [][]byte{addition}
	update := map[string][][]byte{"log": log}
	c.Assert(Write(getThread(1).Update(update)), IsNil)
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

func (*DBSuite) TestRegisterAccount(c *C) {
	const id = "123"
	hash := []byte{1, 2, 3}
	user := auth.User{
		ID:       id,
		Password: hash,
		Rigths:   []auth.Right{},
		Sessions: []auth.Session{},
	}

	// New user
	c.Assert(RegisterAccount(id, hash), IsNil)
	var res auth.User
	c.Assert(One(GetAccount(id), &res), IsNil)
	c.Assert(res, DeepEquals, user)

	// User name already registered
	c.Assert(RegisterAccount(id, hash), ErrorMatches, "user name already taken")
}

func (*DBSuite) TestNonExistantUserGetHash(c *C) {
	_, err := GetLoginHash("123")
	c.Assert(err, Equals, r.ErrEmptyResult)
}

func (*DBSuite) TestGetLoginHash(c *C) {
	const id = "123"
	hash := []byte{1, 2, 3}
	user := auth.User{
		ID:       id,
		Password: hash,
	}
	c.Assert(Write(r.Table("accounts").Insert(user)), IsNil)

	res, err := GetLoginHash(id)
	c.Assert(err, IsNil)
	c.Assert(res, DeepEquals, hash)
}
