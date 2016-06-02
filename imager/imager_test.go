package imager

import (
	"io/ioutil"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
	. "gopkg.in/check.v1"
)

func (*DB) TestVerifyImageFormat(c *C) {
	samples := map[string]bool{
		"jpeg": true,
		"gif":  true,
		"png":  true,
		"webm": false,
	}
	var postID int64
	for ext, shouldPass := range samples {
		file := openFile("sample."+ext, c)
		defer file.Close()
		err := verifyImage(file, postID)
		postID++
		if shouldPass {
			c.Assert(err, IsNil)
		} else {
			c.Assert(err, ErrorMatches, "Unsupported image format: .*")
		}
	}

	// Failure to decode
	file := openFile("sample.txt", c)
	defer file.Close()
	err := verifyImage(file, postID)
	c.Assert(err, ErrorMatches, "Error decoding image: .*")
}

func (*DB) TestVerifyDimentions(c *C) {
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

	c.Assert(verifyImage(tooTall, 1), ErrorMatches, "Image too tall")
	c.Assert(verifyImage(tooWide, 2), ErrorMatches, "Image too wide")
	c.Assert(verifyImage(pass, 3), IsNil)
}

func (*DB) TestDupDetection(c *C) {
	sample := openFile("sample.jpeg", c)
	defer sample.Close()
	c.Assert(verifyImage(sample, 1), IsNil)
	_, err := sample.Seek(0, 0)
	c.Assert(err, IsNil)
	c.Assert(verifyImage(sample, 2), ErrorMatches, "Duplicate image of post 1")
}

func (*DB) TestImageProcessing(c *C) {
	samples := map[string]uint8{
		"jpeg": jpeg,
		"gif":  gif,
		"png":  png,
	}
	for ext, filetype := range samples {
		file := openFile("sample."+ext, c)
		defer file.Close()
		img := &types.ProtoImage{
			ImageCommon: types.ImageCommon{
				FileType: filetype,
			},
		}

		large, small, err := processImage(file, int64(filetype)+20, img)
		c.Assert(err, IsNil)
		smallBuf, err := ioutil.ReadAll(small)
		c.Assert(err, IsNil)
		largeBuf, err := ioutil.ReadAll(large)
		c.Assert(err, IsNil)
		c.Assert(len(largeBuf) > len(smallBuf), Equals, true)
	}
}
