package imager

import (
	"github.com/bakape/meguca/config"
	. "gopkg.in/check.v1"
	"io/ioutil"
)

func (*DB) TestVerifyImageFormat(c *C) {
	samples := map[string]bool{
		"jpeg": true,
		"gif":  true,
		"png":  true,
		"webm": false,
	}
	var postID uint64
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

func (*Imager) TestFileHashing(c *C) {
	img := &ProtoImage{}
	hashFile([]byte{1, 2, 3}, img)
	c.Assert(img.SHA1, Equals, "7037807198c22a7d2b0807371d763779a84fdfcf")
	c.Assert(img.MD5, Equals, "5289df737df57326fcdd22597afb1fac")
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
		img := &ProtoImage{
			fileType: filetype,
			PostID:   uint64(filetype) + 20,
		}
		c.Assert(processImage(file, img), IsNil)
		c.Assert(len(img.SHA1) > len(img.MD5), Equals, true)
		small, err := ioutil.ReadAll(img.Thumbnail)
		c.Assert(err, IsNil)
		large, err := ioutil.ReadAll(img.SharpThumbnail)
		c.Assert(err, IsNil)
		c.Assert(len(large) > len(small), Equals, true)
	}
}
