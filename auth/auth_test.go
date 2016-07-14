package auth

import (
	"testing"

	"github.com/bakape/meguca/config"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Auth struct{}

var _ = Suite(&Auth{})

func (*Auth) TestLookupIdent(c *C) {
	const ip = "::1"
	ident := Ident{IP: ip}
	c.Assert(LookUpIdent(ip), DeepEquals, ident)
}

func (*Auth) TestIsBoard(c *C) {
	config.Set(config.Configs{
		Boards: []string{"a", ":^)"},
	})

	// Board exists
	c.Assert(IsBoard("a"), Equals, true)

	// Non-alphanumeric board name
	c.Assert(IsBoard(`:%5E%29`), Equals, true)

	// Board doesn't exist
	c.Assert(IsBoard("b"), Equals, false)

	// /all/ board
	c.Assert(IsBoard("all"), Equals, true)
}
