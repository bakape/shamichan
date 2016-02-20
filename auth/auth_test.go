package auth

import (
	"github.com/bakape/meguca/config"
	. "gopkg.in/check.v1"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type Auth struct{}

var _ = Suite(&Auth{})

func (*Auth) TestCheckAuth(c *C) {
	config.Config = config.Server{}
	config.Config.Staff.Classes = make(map[string]config.StaffClass, 1)
	config.Config.Staff.Classes["admin"] = config.StaffClass{
		Rights: map[string]bool{
			"canFoo": true,
			"canBar": false,
		},
	}

	// Staff with rights
	ident := Ident{Auth: "admin"}
	c.Assert(Check("canFoo", ident), Equals, true)

	// Staff without rights
	c.Assert(Check("canBar", ident), Equals, false)
	c.Assert(Check("canBaz", ident), Equals, false)

	// Non-existant staff
	ident = Ident{Auth: "butler"}
	c.Assert(Check("canFoo", ident), Equals, false)

	// Not staff
	ident = Ident{}
	c.Assert(Check("canFoo", ident), Equals, false)
}

func (*Auth) TestLookupIdent(c *C) {
	const ip = "::1"
	ident := Ident{IP: ip}
	c.Assert(LookUpIdent(ip), DeepEquals, ident)
}

func (*Auth) TestCanAccessBoard(c *C) {
	config.Config = config.Server{}
	config.Config.Boards.Enabled = []string{"a"}
	ident := Ident{}

	// Board exists
	c.Assert(CanAccessBoard("a", ident), Equals, true)

	// Board doesn't exist
	c.Assert(CanAccessBoard("b", ident), Equals, false)

	// /all/ board
	c.Assert(CanAccessBoard("all", ident), Equals, true)

	// Banned
	ident = Ident{Banned: true}
	c.Assert(CanAccessBoard("a", ident), Equals, false)
}
