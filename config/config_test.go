package config

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Tests struct{}

var _ = Suite(&Tests{})

func (*Tests) TestSetGet(c *C) {
	conf := Configs{}
	conf.Hats = true
	Set(conf)
	c.Assert(Get(), DeepEquals, &conf)
}

func (*Tests) TestSetGetClient(c *C) {
	std := []byte{1, 2, 3}
	hash := "foo"
	SetClient(std, hash)
	json, jsonHash := GetClient()
	c.Assert(json, DeepEquals, std)
	c.Assert(jsonHash, Equals, hash)
}

func (*Tests) TestSetGetBoards(c *C) {
	std := []string{"a"}
	SetBoards(std)
	c.Assert(GetBoards(), DeepEquals, std)
}
