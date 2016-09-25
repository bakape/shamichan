package db

import (
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*Tests) TestValidateOP(c *C) {
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

func (*Tests) TestGetThread(c *C) {
	c.Assert(getThread(1).String(), Equals, `r.Table("threads").Get(1)`)
}

func (*Tests) TestPostCounter(c *C) {
	std := infoDocument{
		Document: Document{"info"},
		PostCtr:  1,
	}
	c.Assert(Write(r.Table("main").Insert(std)), IsNil)

	count, err := PostCounter()
	c.Assert(err, IsNil)
	c.Assert(count, Equals, int64(1))
}

func (*Tests) TestBoardCounter(c *C) {
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

func (*Tests) TestThreadCounter(c *C) {
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

func (*Tests) TestRegisterAccount(c *C) {
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

func (*Tests) TestNonExistantUserGetHash(c *C) {
	_, err := GetLoginHash("123")
	c.Assert(err, Equals, r.ErrEmptyResult)
}

func (*Tests) TestGetLoginHash(c *C) {
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

func (*Tests) TestGetImage(c *C) {
	c.Assert(GetImage("123").String(), Equals, `r.Table("images").Get("123")`)
}

func (*Tests) TestGetBoardConfig(c *C) {
	const q = `r.Table("boards").Get("a").Without("created")`
	c.Assert(GetBoardConfig("a").String(), Equals, q)
}

func (*Tests) TestReservePostID(c *C) {
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

func (*Tests) TestIncrementBoardCounter(c *C) {
	c.Assert(Write(r.Table("main").Insert(Document{"boardCtrs"})), IsNil)

	// Check both a fresh board counter and incrementing an existing one
	for i := int64(1); i <= 2; i++ {
		c.Assert(IncrementBoardCounter("a"), IsNil)
		ctr, err := BoardCounter("a")
		c.Assert(err, IsNil)
		c.Assert(ctr, Equals, i)
	}
}
