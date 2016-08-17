package imager

import (
	"fmt"
	"image"
	jpegLib "image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/Soreil/imager"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Imager struct{}

var (
	_ = Suite(&Imager{})

	// Resulting dimentions after thumbnailing samples
	jpegDims = [4]uint16{0x43c, 0x371, 0x96, 0x79}
	pngDims  = [4]uint16{0x500, 0x2d0, 0x96, 0x54}
	gifDims  = [4]uint16{0x248, 0x2d0, 0x7a, 0x96}
)

func (d *Imager) SetUpSuite(c *C) {
	isTest = true
	db.DBName = db.UniqueDBName()
	c.Assert(db.Connect(), IsNil)
	c.Assert(db.InitDB(), IsNil)

	for _, dir := range [...]string{"src", "thumb"} {
		path := filepath.FromSlash("images/" + dir)
		c.Assert(os.MkdirAll(path, 0770), IsNil)
	}
}

func (d *Imager) SetUpTest(c *C) {
	config.Set(config.Configs{
		MaxHeight: 10000,
		MaxWidth:  10000,
		MaxSize:   10,
	})
}

func (d *Imager) TearDownTest(c *C) {
	// Clear DB tables
	for _, table := range db.AllTables {
		c.Assert(db.Write(r.Table(table).Delete()), IsNil)
	}

	// Clear image asset folders
	for _, dir := range [...]string{"src", "thumb"} {
		path := filepath.FromSlash("images/" + dir)
		dirh, err := os.Open(path)
		c.Assert(err, IsNil)
		defer dirh.Close()
		files, err := dirh.Readdirnames(-1)
		c.Assert(err, IsNil)
		for _, file := range files {
			path := fmt.Sprintf("images/%s/%s", dir, file)
			path = filepath.FromSlash(path)
			c.Assert(os.Remove(path), IsNil)
		}
	}
}

func (d *Imager) TearDownSuite(c *C) {
	c.Assert(db.Write(r.DBDrop(db.DBName)), IsNil)
	c.Assert(db.RSession.Close(), IsNil)
	c.Assert(os.RemoveAll("img"), IsNil)
}

func (*Imager) TestInitImager(c *C) {
	config.Set(config.Configs{
		JPEGQuality: 90,
		PNGQuality:  20,
	})
	InitImager()
	c.Assert(imager.JPEGOptions, Equals, jpegLib.Options{Quality: 90})
	c.Assert(imager.PNGQuantization, Equals, 20)
}

func (*Imager) TestVerifyDimentions(c *C) {
	config.Set(config.Configs{
		MaxWidth:  2000,
		MaxHeight: 2000,
	})

	samples := [...]struct {
		name string
		err  error
		dims [4]uint16
	}{
		{"too wide.jpg", errTooWide, [4]uint16{2001, 720, 0, 0}},
		{"too tall.jpg", errTooTall, [4]uint16{1280, 2001, 0, 0}},
		{"sample.jpg", nil, [4]uint16{1084, 881, 0, 0}},
	}

	for _, s := range samples {
		file := openFile(s.name, c)
		defer file.Close()
		img, _, err := image.Decode(file)
		c.Assert(err, IsNil)
		dims, err := verifyDimentions(img)
		c.Assert(err, Equals, s.err)
		c.Assert(dims, Equals, s.dims)
	}
}

func (*Imager) TestImageProcessing(c *C) {
	samples := [...]struct {
		ext  string
		dims [4]uint16
	}{
		{"jpg", jpegDims},
		{"png", pngDims},
		{"gif", gifDims},
	}

	for _, s := range samples {
		thumb, dims, err := processImage(readSample("sample."+s.ext, c))
		c.Assert(err, IsNil)
		assertThumbnail(thumb, c)
		c.Assert(dims, Equals, s.dims)
	}
}

// How do we assert a thumbnail?
func assertThumbnail(thumb []byte, c *C) {
	c.Assert(len(thumb) > 100, Equals, true)
}
