package config

import (
	"encoding/json"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Tests struct{}

var _ = Suite(&Tests{})

func (*Tests) SetUpTest(c *C) {
	global = nil
	clientJSON = nil
	boardConfigs = nil
	hash = ""
}

func (*Tests) TestSetGet(c *C) {
	conf := Configs{}
	conf.Hats = true
	c.Assert(Set(conf), IsNil)
	c.Assert(Get(), DeepEquals, &conf)
	json, hash := GetClient()
	c.Assert(json, NotNil)
	c.Assert(hash, Not(Equals), "")
}

func (*Tests) TestSetGetClient(c *C) {
	std := []byte{1, 2, 3}
	hash := "foo"
	SetClient(std, hash)
	json, jsonHash := GetClient()
	c.Assert(json, DeepEquals, std)
	c.Assert(jsonHash, Equals, hash)
}

func (*Tests) TestMarshalSpoilers(c *C) {
	data, err := json.Marshal(spoilers{1, 2, 3})
	c.Assert(err, IsNil)
	c.Assert(string(data), Equals, `[1,2,3]`)
}
