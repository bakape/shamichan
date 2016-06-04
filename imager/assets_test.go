package imager

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"
)

func (*Imager) TestGetFilePaths(c *C) {
	webm := getFilePaths("jingai", webm)
	jpeg := getFilePaths("modoki", jpeg)
	checks := [...]struct {
		got, expected string
	}{
		{webm[0], "./img/src/jingai.webm"},
		{webm[1], "./img/thumb/jingai.png"},
		{webm[2], "./img/mid/jingai.png"},
		{jpeg[0], "./img/src/modoki.jpg"},
		{jpeg[1], "./img/thumb/modoki.jpg"},
		{jpeg[2], "./img/mid/modoki.jpg"},
	}
	for _, check := range checks {
		c.Assert(check.got, Equals, filepath.FromSlash(check.expected))
	}
}

func (*Imager) TestDeleteAssets(c *C) {
	samples := [...]struct {
		name     string
		fileType uint8
	}{
		{"foo", jpeg},
		{"bar", png},
	}

	// Create all sample files
	for _, sample := range samples {
		for _, path := range getFilePaths(sample.name, sample.fileType) {
			file, err := os.Create(path)
			c.Assert(err, IsNil)
			file.Close()
		}
	}

	// Delete them and check, if deleted
	for _, sample := range samples {
		c.Assert(deleteAssets(sample.name, sample.fileType), IsNil)

		for _, path := range getFilePaths(sample.name, sample.fileType) {
			_, err := os.Stat(path)
			c.Assert(err, NotNil)
			c.Assert(os.IsNotExist(err), Equals, true)
		}
	}
}

func (*Imager) TestWriteFile(c *C) {
	std := []byte{1, 0, 0, 3, 2}
	r := bytes.NewReader(std)
	path := filepath.FromSlash("img/src/write_test")
	c.Assert(writeFile(path, r), IsNil)
	defer os.Remove(path)

	buf, err := ioutil.ReadFile(path)
	c.Assert(err, IsNil)
	c.Assert(buf, DeepEquals, std)
}

func (*Imager) TestWriteAssets(c *C) {
	const (
		name     = "foo"
		fileType = jpeg
	)
	std := [...][]byte{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	var rs [3]io.Reader
	for i := range std {
		rs[i] = bytes.NewReader(std[i])
	}

	c.Assert(writeAssets(name, fileType, rs[0], rs[1], rs[2]), IsNil)
	for i, path := range getFilePaths(name, fileType) {
		buf, err := ioutil.ReadFile(path)
		c.Assert(err, IsNil)
		c.Assert(buf, DeepEquals, std[i])
	}
}
