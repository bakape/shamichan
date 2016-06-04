package imager

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"testing"

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

var _ = Suite(&Imager{})

func (d *Imager) SetUpSuite(c *C) {
	d.dbName = db.UniqueDBName()
	c.Assert(db.Connect(""), IsNil)
	c.Assert(db.InitDB(d.dbName), IsNil)

	for _, dir := range [...]string{"src", "thumb", "mid"} {
		path := filepath.FromSlash("./img/" + dir)
		c.Assert(os.MkdirAll(path, 0770), IsNil)
	}
}

func (d *Imager) SetUpTest(c *C) {
	conf := config.ServerConfigs{}
	conf.Images.Max.Height = 10000
	conf.Images.Max.Width = 10000
	conf.Images.Max.Size = 1024
	conf.Images.Spoilers = []uint8{1, 2}
	config.Set(conf)
}

func (d *Imager) TearDownTest(c *C) {
	for _, table := range db.AllTables {
		c.Assert(db.DB(r.Table(table).Delete()).Exec(), IsNil)
	}
	websockets.Clients.Clear()
}

func (d *Imager) TearDownSuite(c *C) {
	c.Assert(db.DB(r.DBDrop(d.dbName)).Exec(), IsNil)
	c.Assert(db.RSession.Close(), IsNil)
	c.Assert(os.RemoveAll("img"), IsNil)
}

func (*Imager) TestVerifyImageFormat(c *C) {
	samples := map[string]bool{
		"jpg":  true,
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
	err := verifyImage(file)
	c.Assert(err, ErrorMatches, "Error decoding image: .*")
}

func (*Imager) TestVerifyDimentions(c *C) {
	conf := config.ServerConfigs{}
	conf.Images.Max.Width = 2000
	conf.Images.Max.Height = 2000
	config.Set(conf)

	tooWide := openFile("too wide.jpg", c)
	tooTall := openFile("too tall.jpg", c)
	pass := openFile("sample.jpg", c)
	defer func() {
		tooTall.Close()
		tooWide.Close()
		pass.Close()
	}()

	c.Assert(verifyImage(tooTall), ErrorMatches, "Image too tall")
	c.Assert(verifyImage(tooWide), ErrorMatches, "Image too wide")
	c.Assert(verifyImage(pass), IsNil)
}

func (*Imager) TestImageProcessing(c *C) {
	for _, ext := range [...]string{"jpg", "gif", "png"} {
		file := openFile("sample."+ext, c)
		defer file.Close()

		large, small, err := processImage(file)
		c.Assert(err, IsNil)
		smallBuf, err := ioutil.ReadAll(small)
		c.Assert(err, IsNil)
		largeBuf, err := ioutil.ReadAll(large)
		c.Assert(err, IsNil)
		c.Assert(len(largeBuf) > len(smallBuf), Equals, true)
	}
}
