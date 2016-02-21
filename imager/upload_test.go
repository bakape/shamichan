package imager

import (
	"github.com/bakape/meguca/config"
	. "gopkg.in/check.v1"
	"mime/multipart"
	"os"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type Imager struct{}

var _ = Suite(&Imager{})

func (*Imager) TestIsValidSpoiler(c *C) {
	config.Config = config.Server{}
	config.Config.Images.Spoilers = []uint8{1, 2}
	c.Assert(isValidSpoiler(8), Equals, false)
	c.Assert(isValidSpoiler(1), Equals, true)
}

func (*Imager) TestDetectFileType(c *C) {
	// Supported file types
	types := [...]string{".jpg", ".gif", ".png", ".webm", ".pdf"}
	for _, ext := range types {
		f := openFile("./test/uploads/sample"+ext, c)
		t, err := detectFileType(f)
		c.Assert(err, IsNil)
		c.Assert(t, Equals, ext)
	}
}

func openFile(path string, c *C) multipart.File {
	f, err := os.Open(path)
	c.Assert(err, IsNil)
	return f
}
