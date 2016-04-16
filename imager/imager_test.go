package imager

import (
	"github.com/bakape/meguca/config"
	. "gopkg.in/check.v1"
)

func (*Imager) TestVerifyImageFormat(c *C) {
	conf := config.ServerConfigs{}
	conf.Images.Max.Height = 10000
	conf.Images.Max.Width = 10000
	config.Set(conf)
	samples := map[string]bool{
		"jpeg": true,
		"gif":  true,
		"png":  true,
		"webm": false,
	}
	for ext, shouldPass := range samples {
		file := openFile("sample."+ext, c)
		defer file.Close()
		err := verifyImage(file)
		if shouldPass {
			c.Assert(err, IsNil)
		} else {
			c.Assert(err, ErrorMatches, "Unsupported image format: .*")
		}
	}

	// Failure to decode
	file := openFile("sample.txt", c)
	defer file.Close()
	c.Assert(verifyImage(file), ErrorMatches, "Error decoding image: .*")
}

func (*Imager) TestVerifyDimentions(c *C) {
	conf := config.ServerConfigs{}
	conf.Images.Max.Width = 2000
	conf.Images.Max.Height = 2000
	config.Set(conf)

	tooWide := openFile("too wide.jpg", c)
	tooTall := openFile("too tall.jpg", c)
	pass := openFile("sample.jpeg", c)
	defer func() {
		tooTall.Close()
		tooWide.Close()
		pass.Close()
	}()

	c.Assert(verifyImage(tooTall), ErrorMatches, "Image too tall")
	c.Assert(verifyImage(tooWide), ErrorMatches, "Image too wide")
	c.Assert(verifyImage(pass), IsNil)
}
