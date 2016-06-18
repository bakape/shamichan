package auth

import (
	"testing"

	"github.com/bakape/meguca/config"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Auth struct{}

var _ = Suite(&Auth{})

func (*Auth) TestCheckAuth(c *C) {
	conf := config.ServerConfigs{}
	conf.Staff.Classes = make(map[string]config.StaffClass, 1)
	conf.Staff.Classes["admin"] = config.StaffClass{
		Rights: map[string]bool{
			"canFoo": true,
			"canBar": false,
		},
	}
	config.Set(conf)

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

func (*Auth) TestIsBoard(c *C) {
	conf := config.ServerConfigs{}
	conf.Boards.Enabled = []string{"a"}
	config.Set(conf)

	// Board exists
	c.Assert(IsBoard("a"), Equals, true)

	// Board doesn't exist
	c.Assert(IsBoard("b"), Equals, false)

	// /all/ board
	c.Assert(IsBoard("all"), Equals, true)
}
