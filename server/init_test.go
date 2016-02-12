package server

import (
	. "gopkg.in/check.v1"
	"os"
)

type Init struct{}

var _ = Suite(&Init{})

func (i *Init) TestCreateDirs(c *C) {
	createDirs()
	for _, dir := range [...]string{"src", "thumb", "mid"} {
		_, err := os.Stat("./img/" + dir)
		c.Assert(err, IsNil)
	}
	c.Assert(os.RemoveAll("./img"), IsNil)
}
