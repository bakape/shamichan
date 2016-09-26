package assets

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bakape/meguca/types"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&Tests{})

type Tests struct{}

func (*Tests) SetUpSuite(c *C) {
	c.Assert(CreateDirs(), IsNil)
}

func (*Tests) TearDownTest(c *C) {
	c.Assert(ResetDirs(), IsNil)
}

func (*Tests) TearDownSuite(c *C) {
	c.Assert(DeleteDirs(), IsNil)
}

func (*Tests) TestGetFilePaths(c *C) {
	webm := GetFilePaths("jingai", types.WEBM)
	jpeg := GetFilePaths("modoki", types.JPEG)
	checks := [...]struct {
		got, expected string
	}{
		{webm[0], "images/src/jingai.webm"},
		{webm[1], "images/thumb/jingai.png"},
		{jpeg[0], "images/src/modoki.jpg"},
		{jpeg[1], "images/thumb/modoki.jpg"},
	}
	for _, check := range checks {
		c.Assert(check.got, Equals, filepath.FromSlash(check.expected))
	}
}

func (*Tests) TestDeleteAssets(c *C) {
	samples := [...]struct {
		name     string
		fileType uint8
	}{
		{"foo", types.JPEG},
		{"bar", types.PNG},
	}

	// Create all sample files
	for _, sample := range samples {
		for _, path := range GetFilePaths(sample.name, sample.fileType) {
			file, err := os.Create(path)
			c.Assert(err, IsNil)
			file.Close()
		}
	}

	// Delete them and check, if deleted
	for _, sample := range samples {
		c.Assert(Delete(sample.name, sample.fileType), IsNil)

		for _, path := range GetFilePaths(sample.name, sample.fileType) {
			_, err := os.Stat(path)
			c.Assert(err, NotNil)
			c.Assert(os.IsNotExist(err), Equals, true)
		}
	}
}

func (*Tests) TestDeleteMissingAssets(c *C) {
	c.Assert(Delete("akari", types.PNG), IsNil)
}

func (*Tests) TestWriteFile(c *C) {
	std := []byte{1, 0, 0, 3, 2}
	path := filepath.FromSlash("images/src/write_test")
	c.Assert(writeFile(path, std), IsNil)

	buf, err := ioutil.ReadFile(path)
	c.Assert(err, IsNil)
	c.Assert(buf, DeepEquals, std)
}

func (*Tests) TestWriteAssets(c *C) {
	const (
		name     = "foo"
		fileType = types.JPEG
	)
	std := [...][]byte{
		{1, 2, 3},
		{4, 5, 6},
	}

	c.Assert(Write(name, fileType, std[0], std[1]), IsNil)
	for i, path := range GetFilePaths(name, fileType) {
		buf, err := ioutil.ReadFile(path)
		c.Assert(err, IsNil)
		c.Assert(buf, DeepEquals, std[i])
	}
}
