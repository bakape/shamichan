package imager

import (
	. "gopkg.in/check.v1"
	"path/filepath"
)

func (*Imager) TestVerifyWebm(c *C) {
	path := filepath.FromSlash("test/sample.webm")
	file := openFile(path, c)
	c.Assert(verifyWebm(file), IsNil)
}
