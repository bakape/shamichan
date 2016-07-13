package config

import (
	"strings"
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
func (*Tests) TestMarshalPublicBoardJSON(c *C) {
	b := BoardConfigs{
		CodeTags: true,
		Spoiler:  "foo.png",
		Title:    "Animu",
	}
	std := `
{
	"codeTags":true,
	"spoiler":"foo.png",
	"title":"Animu",
	"notice":""
}`
	std = strings.Replace(std, "\t", "", -1)
	std = strings.Replace(std, "\n", "", -1)

	data, err := b.MarshalPublicJSON()
	c.Assert(err, IsNil)
	c.Assert(string(data), Equals, std)
}
