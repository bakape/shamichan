package imager

import (
	"bytes"
	"github.com/bakape/meguca/config"
	. "gopkg.in/check.v1"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type Imager struct{}

var _ = Suite(&Imager{})

func (*Imager) TestExtractSpoiler(c *C) {
	config.Config = config.Server{}
	config.Config.Images.Spoilers = []uint8{1, 2}

	// No spoiler
	body, w := newMultiWriter()
	req := newRequest(c, body, w)
	sp, err := extractSpoiler(req)
	c.Assert(err, IsNil)
	c.Assert(sp, Equals, uint8(0))

	// Invalid spoiler
	body, w = newMultiWriter()
	c.Assert(w.WriteField("spoiler", "shibireru darou"), IsNil)
	req = newRequest(c, body, w)
	sp, err = extractSpoiler(req)
	c.Assert(err, ErrorMatches, `Invalid spoiler ID: shibireru darou`)

	// Not an enabled spoiler
	body, w = newMultiWriter()
	c.Assert(w.WriteField("spoiler", "10"), IsNil)
	req = newRequest(c, body, w)
	sp, err = extractSpoiler(req)
	c.Assert(err, ErrorMatches, `Invalid spoiler ID: 10`)

	// Valid spoiler
	body, w = newMultiWriter()
	c.Assert(w.WriteField("spoiler", "1"), IsNil)
	req = newRequest(c, body, w)
	sp, err = extractSpoiler(req)
	c.Assert(err, IsNil)
	c.Assert(sp, Equals, uint8(1))
}

func (*Imager) TestIsValidSpoiler(c *C) {
	config.Config = config.Server{}
	config.Config.Images.Spoilers = []uint8{1, 2}
	c.Assert(isValidSpoiler(8), Equals, false)
	c.Assert(isValidSpoiler(1), Equals, true)
}

var extensions = map[string]int{
	"jpg":  jpeg,
	"png":  png,
	"gif":  gif,
	"webm": webm,
	"pdf":  pdf,
}

func (*Imager) TestDetectFileType(c *C) {
	// Supported file types
	for ext, code := range extensions {
		f := openFile(filepath.FromSlash("./test/uploads/sample."+ext), c)
		t, err := detectFileType(f)
		c.Assert(err, IsNil)
		c.Assert(t, Equals, code)
	}
}

func openFile(path string, c *C) multipart.File {
	f, err := os.Open(path)
	c.Assert(err, IsNil)
	return f
}

func newMultiWriter() (*bytes.Buffer, *multipart.Writer) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	return body, writer
}

func newRequest(c *C, body io.Reader, w *multipart.Writer) *http.Request {
	req, err := http.NewRequest("PUT", "/", body)
	c.Assert(err, IsNil)
	c.Assert(w.Close(), IsNil)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}
