package imager

import (
	. "gopkg.in/check.v1"
)

func (*Imager) TestVerifyWebm(c *C) {
	file := openFile("sample.webm", c)
	defer file.Close()
	c.Assert(verifyWebm(file), IsNil)
}
