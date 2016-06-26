package db

import (
	"time"

	"github.com/bakape/meguca/auth"
	r "github.com/dancannon/gorethink"

	. "gopkg.in/check.v1"
)

func (*DBSuite) TestSessionCleanup(c *C) {
	expired := time.Now().Add(-time.Hour)
	samples := []auth.User{
		{
			ID: "1",
			Sessions: []auth.Session{
				{
					Token:   "foo",
					Expires: expired,
				},
				{
					Token:   "bar",
					Expires: time.Now().Add(time.Hour),
				},
			},
		},
		{
			ID: "2",
			Sessions: []auth.Session{
				{
					Token:   "baz",
					Expires: expired,
				},
			},
		},
	}
	c.Assert(Write(r.Table("accounts").Insert(samples)), IsNil)

	expireUserSessions()

	var res1 []auth.Session
	c.Assert(All(GetAccount("1").Field("sessions"), &res1), IsNil)
	c.Assert(len(res1), Equals, 1)
	c.Assert(res1[0].Token, Equals, "bar")

	var res2 []auth.Session
	c.Assert(All(GetAccount("2").Field("sessions"), &res1), IsNil)
	c.Assert(res2, DeepEquals, []auth.Session(nil))
}
