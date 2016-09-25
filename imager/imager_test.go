package imager

import (
	"image"
	jpegLib "image/jpeg"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/Soreil/imager"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager/assets"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Imager struct{}

var _ = Suite(&Imager{})

func (d *Imager) SetUpSuite(c *C) {
	assetRoot = "testdata"
	db.DBName = db.UniqueDBName()
	c.Assert(db.Connect(), IsNil)
	c.Assert(db.InitDB(), IsNil)
	c.Assert(assets.CreateDirs(), IsNil)
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
	c.Assert(assets.ResetDirs(), IsNil)
}

func (d *Imager) TearDownSuite(c *C) {
	c.Assert(db.Write(r.DBDrop(db.DBName)), IsNil)
	c.Assert(db.RSession.Close(), IsNil)
	c.Assert(assets.DeleteDirs(), IsNil)
}

func readSample(name string, c *C) []byte {
	path := filepath.Join("testdata", name)
	data, err := ioutil.ReadFile(path)
	c.Assert(err, IsNil)
	return data
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
		{"jpg", assets.StdDims["jpeg"]},
		{"png", assets.StdDims["png"]},
		{"gif", assets.StdDims["gif"]},
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
