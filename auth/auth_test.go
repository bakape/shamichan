package auth

import (
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/bakape/meguca/config"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Tests struct{}

var _ = Suite(&Tests{})

func (*Tests) SetUpTest(_ *C) {
	config.Set(config.Configs{})
	IsReverseProxied = false
	ReverseProxyIP = ""
}

func (*Tests) TestIsBoard(c *C) {
	config.Set(config.Configs{
		Boards: []string{"a", ":^)"},
	})

	samples := [...]struct {
		in      string
		isBoard bool
	}{
		{"a", true},   // Board exists
		{"b", false},  // Board doesn't exist
		{"all", true}, // /all/ board
	}

	for _, s := range samples {
		c.Assert(IsBoard(s.in), Equals, s.isBoard)
	}
}

func (*Tests) TestLookupIdentNoReverseProxy(c *C) {
	const ip = "::1"
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = ip
	std := Ident{IP: ip}
	c.Assert(LookUpIdent(req), DeepEquals, std)
}

func (*Tests) TestGetIP(c *C) {
	const (
		ip             = "207.178.71.93"
		reverseProxyIP = "162.30.251.246"
	)
	IsReverseProxied = true
	ReverseProxyIP = reverseProxyIP

	samples := [...]struct {
		xff, out string
	}{
		{"10.121.169.19", "10.121.169.19"},
		{"", ip},
		{"notip, nope", ip},
		{"105.124.243.122, 10.168.239.157, 127.0.0.1, ::1", "10.168.239.157"},
		{"105.124.243.122," + reverseProxyIP, "105.124.243.122"},
	}

	for _, s := range samples {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = ip
		if s.xff != "" {
			req.Header.Set("X-Forwarded-For", s.xff)
		}
		c.Assert(GetIP(req), Equals, s.out)
	}
}

func (*Tests) TestBcryptHash(c *C) {
	const (
		password = "123456"
	)
	hash, err := BcryptHash(password, 8)
	c.Assert(err, IsNil)

	// Mismatch
	err = BcryptCompare(password+"1", hash)
	c.Assert(err, Equals, bcrypt.ErrMismatchedHashAndPassword)

	// Correct
	err = BcryptCompare(password, hash)
	c.Assert(err, IsNil)
}

func (*Tests) TestRandomID(c *C) {
	hash, err := RandomID(32)
	c.Assert(err, IsNil)
	c.Assert(hash, Matches, "^.{43}$")
}
