package db

import (
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

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
	std := Document{"boardCtrs"}
	c.Assert(Write(r.Table("main").Insert(std)), IsNil)

	count, err := BoardCounter("a")
	c.Assert(err, IsNil)
	c.Assert(count, Equals, int64(0))

	update := map[string]int{"a": 1}
	c.Assert(Write(GetMain("boardCtrs").Update(update)), IsNil)

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
	read := make(chan [][]byte)
	closer := make(chan struct{})
	initial, err := StreamUpdates(1, read, closer)
	c.Assert(err, IsNil)
	c.Assert(initial, DeepEquals, [][]byte{})

	log := [][]byte{[]byte{1, 0, 0, 3, 2}}
	update := map[string][][]byte{"log": log}
	c.Assert(Write(getThread(1).Update(update)), IsNil)
	c.Assert(<-read, DeepEquals, log)
	close(closer)

	// Existing data
	read = make(chan [][]byte)
	closer = make(chan struct{})
	initial, err = StreamUpdates(1, read, closer)
	c.Assert(err, IsNil)
	c.Assert(initial, DeepEquals, log)
	close(closer)
}

func (*DBSuite) TestRegisterAccount(c *C) {
	const id = "123"
	hash := []byte{1, 2, 3}
	user := auth.User{
		ID:       id,
		Password: hash,
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

func (*DBSuite) TestGetImage(c *C) {
	c.Assert(GetImage("123").String(), Equals, `r.Table("images").Get("123")`)
}

func (*DBSuite) TestGetBoardConfig(c *C) {
	c.Assert(GetBoardConfig("a").String(), Equals, `r.Table("boards").Get("a")`)
}

func (*DBSuite) TestReservePostID(c *C) {
	info := map[string]interface{}{
		"id":      "info",
		"postCtr": 0,
	}
	c.Assert(Write(r.Table("main").Insert(info)), IsNil)

	for i := int64(1); i <= 2; i++ {
		id, err := ReservePostID()
		c.Assert(err, IsNil)
		c.Assert(id, Equals, i)
	}
}

func (*DBSuite) TestIncrementBoardCounter(c *C) {
	c.Assert(Write(r.Table("main").Insert(Document{"boardCtrs"})), IsNil)

	// Check both a fresh board counter and incrementing an existing one
	for i := int64(1); i <= 2; i++ {
		c.Assert(IncrementBoardCounter("a"), IsNil)
		ctr, err := BoardCounter("a")
		c.Assert(err, IsNil)
		c.Assert(ctr, Equals, i)
	}
}
