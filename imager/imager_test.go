package imager

import (
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"path/filepath"

	jpegLib "image/jpeg"
	"testing"

	"github.com/Soreil/imager"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"

	"github.com/bakape/meguca/server/websockets"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Imager struct {
	dbName           string
	perceptualCloser chan struct{}
}

var (
	_ = Suite(&Imager{})

	// Resulting dimentions after thumbnailing samples
	jpegDims = [4]uint16{1084, 881, 125, 101}
	pngDims  = [4]uint16{1280, 720, 125, 70}
	gifDims  = [4]uint16{584, 720, 101, 125}
)

func (d *Imager) SetUpSuite(c *C) {
	d.dbName = db.UniqueDBName()
	c.Assert(db.Connect(""), IsNil)
	c.Assert(db.InitDB(d.dbName), IsNil)

	for _, dir := range [...]string{"src", "thumb"} {
		path := filepath.FromSlash("img/" + dir)
		c.Assert(os.MkdirAll(path, 0770), IsNil)
	}
}

func (d *Imager) SetUpTest(c *C) {
	conf := config.ServerConfigs{}
	conf.Images.Max.Height = 10000
	conf.Images.Max.Width = 10000
	conf.Images.Max.Size = 1024 * 1024 * 10
	conf.Images.Spoilers = []uint8{1, 2}
	config.Set(conf)
}

func (d *Imager) TearDownTest(c *C) {
	// Clear DB tables
	for _, table := range db.AllTables {
		c.Assert(db.DB(r.Table(table).Delete()).Exec(), IsNil)
	}

	// Clear synchtonised clients
	websockets.Clients.Clear()

	// Clear image asset folders
	for _, dir := range [...]string{"src", "thumb"} {
		path := filepath.FromSlash("img/" + dir)
		files, err := ioutil.ReadDir(path)
		c.Assert(err, IsNil)
		for _, file := range files {
			path := fmt.Sprintf("img/%s/%s", dir, file.Name())
			path = filepath.FromSlash(path)
			c.Assert(os.Remove(path), IsNil)
		}
	}
}

func (d *Imager) TearDownSuite(c *C) {
	c.Assert(db.DB(r.DBDrop(d.dbName)).Exec(), IsNil)
	c.Assert(db.RSession.Close(), IsNil)
	c.Assert(os.RemoveAll("img"), IsNil)
}

func (*Imager) TestInitImager(c *C) {
	conf := config.ServerConfigs{}
	conf.Images.JpegQuality = 90
	conf.Images.PngQuality = 20
	config.Set(conf)
	InitImager()
	c.Assert(imager.JPEGOptions, Equals, jpegLib.Options{Quality: 90})
	c.Assert(imager.PNGQuantization, Equals, 20)
}

func (*Imager) TestVerifyDimentions(c *C) {
	conf := config.ServerConfigs{}
	conf.Images.Max.Width = 2000
	conf.Images.Max.Height = 2000
	config.Set(conf)

	samples := []struct {
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
	samples := []struct {
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
