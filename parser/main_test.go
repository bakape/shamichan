package parser

import (
	"testing"

	"github.com/bakape/meguca/config"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Tests struct{}

var _ = Suite(&Tests{})

func (*Tests) SetUpTest(c *C) {
	config.Set(config.Configs{})
}

func (*Tests) TestStripPsuedoWhitespace(c *C) {
	samples := []struct {
		in, out string
	}{
		{"normal", "normal"},
		{"h\u2000e\u200fl\u202al\u202fo\u205f\u206f", "hello"},
		{"日本\u2062語", "日本語"},
	}
	for _, s := range samples {
		c.Assert(stripPsuedoWhitespace(s.in), Equals, s.out)
	}
}
