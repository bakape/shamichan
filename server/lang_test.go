package server

import (
	. "gopkg.in/check.v1"
)

type Lang struct{}

var _ = Suite(&Lang{})

func (l *Lang) TestLoadLanguagePack(c *C) {
	config = serverConfigs{}
	config.Lang.Enabled = []string{"en_GB"}
	langRoot = "./test"
	loadLanguagePacks()
	c.Assert(langs["en_GB"].Imager["bar"], Equals, "foo")

}
