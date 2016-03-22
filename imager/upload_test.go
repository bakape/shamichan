package imager

import (
	"bytes"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/server/websockets"
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

func (*Imager) SetUpTest(c *C) {
	config.Config = config.Server{}
}

func (*Imager) TestExtractSpoiler(c *C) {
	config.Config.Images.Spoilers = []uint8{1, 2}

	// No spoiler
	body, w := newMultiWriter()
	sp, err := assertExtraction(c, body, w)
	c.Assert(err, IsNil)
	c.Assert(sp, Equals, uint8(0))

	// Invalid spoiler
	body, w = newMultiWriter()
	c.Assert(w.WriteField("spoiler", "shibireru darou"), IsNil)
	sp, err = assertExtraction(c, body, w)
	c.Assert(err, ErrorMatches, `Invalid spoiler ID: shibireru darou`)

	// Not an enabled spoiler
	body, w = newMultiWriter()
	c.Assert(w.WriteField("spoiler", "10"), IsNil)
	sp, err = assertExtraction(c, body, w)
	c.Assert(err, ErrorMatches, `Invalid spoiler ID: 10`)

	// Valid spoiler
	body, w = newMultiWriter()
	c.Assert(w.WriteField("spoiler", "1"), IsNil)
	sp, err = assertExtraction(c, body, w)
	c.Assert(err, IsNil)
	c.Assert(sp, Equals, uint8(1))
}

func assertExtraction(c *C, b io.Reader, w *multipart.Writer) (uint8, error) {
	req := newRequest(c, b, w)
	c.Assert(req.ParseMultipartForm(512), IsNil)
	return extractSpoiler(req)
}

func (*Imager) TestIsValidSpoiler(c *C) {
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

func (*Imager) TestParseUploadForm(c *C) {
	config.Config.Images.Max.Size = 1024
	headers := map[string]string{}
	fields := map[string]string{}

	// Invalid content-length header
	b, w := newMultiWriter()
	req := newRequest(c, b, w)
	headers["Content-Length"] = "KAWFEE"
	setHeaders(req, headers)
	_, err := parseUploadForm(req)
	c.Assert(err, ErrorMatches, ".* invalid syntax")

	// File too large
	b, w = newMultiWriter()
	req = newRequest(c, b, w)
	headers["Content-Length"] = "1025"
	setHeaders(req, headers)
	_, err = parseUploadForm(req)
	c.Assert(err, ErrorMatches, "File too large")

	// Invalid form
	b, w = newMultiWriter()
	req = newRequest(c, b, w)
	headers["Content-Type"] = "GWEEN TEA"
	headers["Content-Length"] = "1024"
	setHeaders(req, headers)
	_, err = parseUploadForm(req)
	c.Assert(err, NotNil)

	// No client ID
	b, w = newMultiWriter()
	req = newRequest(c, b, w)
	delete(headers, "Content-Type")
	setHeaders(req, headers)
	_, err = parseUploadForm(req)
	c.Assert(err, ErrorMatches, "No client ID specified")

	// Client ID not synchronised with server
	b, w = newMultiWriter()
	fields["id"] = "Rokka"
	writeFields(c, w, fields)
	req = newRequest(c, b, w)
	setHeaders(req, headers)
	_, err = parseUploadForm(req)
	c.Assert(err, ErrorMatches, "Bad client ID: .*")

	// Add client to synced clients map
	cl := &websockets.Client{}
	websockets.Clients.Add(cl)
	fields["id"] = cl.ID
	defer websockets.Clients.Remove(cl.ID)

	// Invalid spoiler
	config.Config.Images.Spoilers = []uint8{1, 2}
	fields["spoiler"] = "12"
	b, w = newMultiWriter()
	writeFields(c, w, fields)
	req = newRequest(c, b, w)
	setHeaders(req, headers)
	_, err = parseUploadForm(req)
	c.Assert(err, ErrorMatches, "Invalid spoiler ID: .*")

	// Success
	delete(fields, "spoiler")
	std := &ProtoImage{
		ClientID: cl.ID,
	}
	b, w = newMultiWriter()
	writeFields(c, w, fields)
	req = newRequest(c, b, w)
	setHeaders(req, headers)
	img, err := parseUploadForm(req)
	c.Assert(err, IsNil)
	c.Assert(img, DeepEquals, std)
}

func setHeaders(req *http.Request, headers map[string]string) {
	for key, val := range headers {
		req.Header.Set(key, val)
	}
}

func writeFields(c *C, w *multipart.Writer, fields map[string]string) {
	for key, val := range fields {
		c.Assert(w.WriteField(key, val), IsNil)
	}
}
