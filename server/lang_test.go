package server

import (
	"github.com/bakape/meguca/config"
	. "gopkg.in/check.v1"
)

type Lang struct{}

var _ = Suite(&Lang{})

func (l *Lang) TestLoadLanguagePack(c *C) {
	config.Config = config.Server{}
	config.Config.Lang.Enabled = []string{"en_GB"}
	langRoot = "./test"
	loadLanguagePacks()
	c.Assert(langs["en_GB"].Imager["bar"], Equals, "foo")

}
