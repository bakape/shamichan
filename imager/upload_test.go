package imager

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*Imager) TestDetectFileType(c *C) {
	// Supported file types
	for code, ext := range types.Extensions {
		f := openFile("sample."+ext, c)
		defer f.Close()
		buf, err := ioutil.ReadAll(f)
		c.Assert(err, IsNil)
		t, err := detectFileType(buf)
		c.Assert(err, IsNil)
		c.Assert(t, Equals, code)
	}
}

func openFile(name string, c *C) *os.File {
	f, err := os.Open(filepath.FromSlash("testdata/" + name))
	c.Assert(err, IsNil)
	return f
}

func newMultiWriter() (*bytes.Buffer, *multipart.Writer) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	return body, writer
}

func newRequest(c *C, body io.Reader, w *multipart.Writer) *http.Request {
	req := httptest.NewRequest("PUT", "/", body)
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

	c.Assert(parseUploadForm(req), ErrorMatches, ".* invalid syntax")
}

func (*Imager) TestUploadTooLarge(c *C) {
	conf := config.Get()
	(*conf).MaxSize = 1
	b, w := newMultiWriter()
	req := newRequest(c, b, w)
	req.Header.Set("Content-Length", "1048577")

	c.Assert(parseUploadForm(req), ErrorMatches, "file too large")
}

func (*Imager) TestInvalidForm(c *C) {
	b, w := newMultiWriter()
	req := newRequest(c, b, w)
	setHeaders(req, map[string]string{
		"Content-Length": "1024",
		"Content-Type":   "GWEEN TEA",
	})

	c.Assert(parseUploadForm(req), NotNil)
}

func (*Imager) TestSuccessfulFormParse(c *C) {
	b, w := newMultiWriter()
	req := newRequest(c, b, w)
	req.Header.Set("Content-Length", "1024")

	c.Assert(parseUploadForm(req), IsNil)
}

func setHeaders(req *http.Request, headers map[string]string) {
	for key, val := range headers {
		req.Header.Set(key, val)
	}
}

func (*Imager) TestWrongFileType(c *C) {
	data := readSample("sample.txt", c)
	code, _, err := newThumbnail(data, types.ImageCommon{})
	c.Assert(err, ErrorMatches, "unsupported file type.*")
	c.Assert(code, Equals, 400)
}

func (*Imager) TestNewThumbnail(c *C) {
	req := newJPEGRequest(c)
	rec := httptest.NewRecorder()
	NewImageUpload(rec, req)
	c.Assert(rec.Code, Equals, 200)

	std := types.ProtoImage{
		ImageCommon: assets.StdJPEG.ImageCommon,
		Posts:       1,
	}
	var img types.ProtoImage
	c.Assert(db.One(r.Table("images").Get(std.SHA1), &img), IsNil)
	c.Assert(img, DeepEquals, std)

	assertImageToken(rec.Body.String(), std.SHA1, assets.StdJPEG.Name, c)
	assertFiles("sample.jpg", std.SHA1, types.JPEG, c)
}

// Assert image file assets were created with the correct paths
func assertFiles(src, id string, fileType uint8, c *C) {
	var (
		paths [3]string
		data  [3][]byte
	)
	paths[0] = filepath.FromSlash("testdata/" + src)
	destPaths := assets.GetFilePaths(id, fileType)
	paths[1], paths[2] = destPaths[0], destPaths[1]

	for i := range paths {
		var err error
		data[i], err = ioutil.ReadFile(paths[i])
		c.Assert(err, IsNil)
	}

	c.Assert(data[0], DeepEquals, data[1])
	c.Assert(len(data[1]) > len(data[2]), Equals, true)
}

func assertImageToken(id, SHA1, name string, c *C) {
	q := r.Table("imageTokens").Get(id).Field("SHA1").Eq(SHA1)
	var isEqual bool
	c.Assert(db.One(q, &isEqual), IsNil)
	c.Assert(isEqual, Equals, true)
}

func (*Imager) TestAPNGThumbnailing(c *C) {
	for _, ext := range [...]string{"png", "apng"} {
		img := types.ImageCommon{
			SHA1: ext,
		}
		data := readSample("sample."+ext, c)

		_, _, err := newThumbnail(data, img)
		c.Assert(err, IsNil)

		c.Assert(getImageRecord(ext, c).APNG, Equals, ext == "apng")
	}
}

func getImageRecord(id string, c *C) (res types.ImageCommon) {
	c.Assert(db.One(db.GetImage(id), &res), IsNil)
	return
}

func (*Imager) TestNoImageUploaded(c *C) {
	b, w := newMultiWriter()
	req := newRequest(c, b, w)
	req.Header.Set("Content-Length", "300792")

	code, _, err := newImageUpload(req)
	c.Assert(err, ErrorMatches, "http: no such file")
	c.Assert(code, Equals, 400)
}

func (*Imager) TestThumbNailReuse(c *C) {
	for i := 0; i < 2; i++ {
		req := newJPEGRequest(c)
		code, _, err := newImageUpload(req)
		c.Assert(err, IsNil)
		c.Assert(code, Equals, 200)

		assertImageRefCount(assets.StdJPEG.SHA1, i+1, c)
	}
}

func assertImageRefCount(id string, count int, c *C) {
	var posts int
	c.Assert(db.One(db.GetImage(id).Field("posts"), &posts), IsNil)
	c.Assert(posts, Equals, count)
}

func newJPEGRequest(c *C) *http.Request {
	var wg sync.WaitGroup
	wg.Add(1)
	b, w := newMultiWriter()

	file, err := w.CreateFormFile("image", assets.StdJPEG.Name)
	c.Assert(err, IsNil)
	_, err = file.Write(readSample(assets.StdJPEG.Name, c))
	c.Assert(err, IsNil)

	req := newRequest(c, b, w)
	req.Header.Set("Content-Length", "300792")

	return req
}

func (*Imager) TestUploadHandler(c *C) {
	req := newJPEGRequest(c)
	rec := httptest.NewRecorder()

	NewImageUpload(rec, req)
	acao := rec.Header().Get("Access-Control-Allow-Origin")
	c.Assert(acao, Equals, config.AllowedOrigin)
	c.Assert(rec.Code, Equals, 200)
}

func (*Imager) TestErrorPassing(c *C) {
	const ip = "::1"
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = ip
	rec := httptest.NewRecorder()

	NewImageUpload(rec, req)
	c.Assert(rec.Code, Equals, 400)
	const errMsg = "strconv.ParseInt: parsing \"\": invalid syntax\n"
	c.Assert(rec.Body.String(), Equals, errMsg)
}
