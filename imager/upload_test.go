package imager

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/server/websockets"
	"github.com/bakape/meguca/types"
	. "gopkg.in/check.v1"
)

func (*Imager) TestExtractSpoiler(c *C) {
	conf := config.ServerConfigs{}
	conf.Images.Spoilers = []uint8{1, 2}
	config.Set(conf)

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
	conf := config.ServerConfigs{}
	conf.Images.Spoilers = []uint8{1, 2}
	config.Set(conf)
	c.Assert(isValidSpoiler(8), Equals, false)
	c.Assert(isValidSpoiler(1), Equals, true)
}

func (*Imager) TestDetectFileType(c *C) {
	// Supported file types
	for code, ext := range extensions {
		f := openFile("sample."+ext, c)
		defer f.Close()
		t, err := detectFileType(f)
		c.Assert(err, IsNil)
		c.Assert(t, Equals, code)
	}
}

func openFile(name string, c *C) *os.File {
	f, err := os.Open(filepath.FromSlash("test/" + name))
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

func (*Imager) TestInvalidContentLengthHeader(c *C) {
	b, w := newMultiWriter()
	req := newRequest(c, b, w)
	setHeaders(req, map[string]string{
		"Content-Length": "KAWFEE",
	})

	_, _, err := parseUploadForm(req)
	c.Assert(err, ErrorMatches, ".* invalid syntax")
}

func (*Imager) TestUploadTooLarge(c *C) {
	b, w := newMultiWriter()
	req := newRequest(c, b, w)
	req.Header.Set("Content-Length", "1025")

	_, _, err := parseUploadForm(req)
	c.Assert(err, ErrorMatches, "File too large")
}

func (*Imager) TestInvalidForm(c *C) {
	b, w := newMultiWriter()
	req := newRequest(c, b, w)
	setHeaders(req, map[string]string{
		"Content-Length": "1024",
		"Content-Type":   "GWEEN TEA",
	})

	_, _, err := parseUploadForm(req)
	c.Assert(err, NotNil)
}

func (*Imager) TestNoClientID(c *C) {
	b, w := newMultiWriter()
	req := newRequest(c, b, w)
	req.Header.Set("Content-Length", "1024")

	_, _, err := parseUploadForm(req)
	c.Assert(err, ErrorMatches, "No client ID specified")
}

func (*Imager) TestInvalidSpoiler(c *C) {
	b, w := newMultiWriter()
	fields := syncClient()
	fields["spoiler"] = "12"
	writeFields(c, w, fields)
	req := newRequest(c, b, w)
	req.Header.Set("Content-Length", "1024")

	_, _, err := parseUploadForm(req)
	c.Assert(err, ErrorMatches, "Invalid spoiler ID: .*")
}

// Add client to synced clients map
func syncClient() map[string]string {
	cl := &websockets.Client{}
	websockets.Clients.Add(cl, "1")
	return map[string]string{"id": cl.ID}
}

func (*Imager) TestSuccessfulFormParse(c *C) {
	b, w := newMultiWriter()
	fields := syncClient()
	fields["spoiler"] = "2"
	writeFields(c, w, fields)
	req := newRequest(c, b, w)
	req.Header.Set("Content-Length", "1024")

	id, spoiler, err := parseUploadForm(req)
	c.Assert(err, IsNil)
	c.Assert(id, Equals, fields["id"])
	c.Assert(spoiler, Equals, uint8(2))
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

func (*Imager) TestPassImage(c *C) {
	img := types.Image{
		ImageCommon: types.ImageCommon{
			File: "123",
		},
	}
	client := new(websockets.Client)
	client.AllocateImage = make(chan types.Image)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Assert(<-client.AllocateImage, DeepEquals, img)
	}()
	passImage(img, client)
	wg.Wait()
}

func (*Imager) TestPassImageTimeout(c *C) {
	oldTimeout := allocationTimeout
	allocationTimeout = time.Second
	defer func() {
		allocationTimeout = oldTimeout
	}()
	client := new(websockets.Client)
	client.AllocateImage = make(chan types.Image)

	const id = "123"
	proto := types.ProtoImage{
		ImageCommon: types.ImageCommon{
			FileType: jpeg,
			File:     id,
		},
		Posts: 2,
	}
	img := types.Image{
		ImageCommon: proto.ImageCommon,
	}
	insertProtoImage(proto, c)

	passImage(img, client)
	assertImageRefCount(img.File, 1, c)
}
