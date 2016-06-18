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
	"time"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/server/websockets"
	"github.com/bakape/meguca/types"
	. "gopkg.in/check.v1"
)

var (
	// JPEG sample image standard struct
	stdJPEG = types.Image{
		ImageCommon: types.ImageCommon{
			SHA1:     "012a2f912c9ee93ceb0ccb8684a29ec571990a94",
			FileType: jpeg,
			Dims:     jpegDims,
			MD5:      "60e41092581f7b329b057b8402caa8a7",
			Size:     300792,
		},
		Imgnm:   "sample.jpg",
		Spoiler: 1,
	}
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
	c.Assert(req.ParseMultipartForm(0), IsNil)
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
		buf, err := ioutil.ReadAll(f)
		c.Assert(err, IsNil)
		t, err := detectFileType(buf)
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
	conf := config.Get()
	(*conf).Images.Max.Size = 1024
	b, w := newMultiWriter()
	req := newRequest(c, b, w)
	req.Header.Set("Content-Length", "1048587")

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
			SHA1: "123",
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
	code, err := passImage(img, client)
	c.Assert(err, IsNil)
	c.Assert(code, Equals, 200)
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
			SHA1:     id,
		},
		Posts: 2,
	}
	img := types.Image{
		ImageCommon: proto.ImageCommon,
	}
	insertProtoImage(proto, c)

	code, err := passImage(img, client)
	c.Assert(err, Equals, errUsageTimeout)
	c.Assert(code, Equals, 408)
	assertImageRefCount(img.SHA1, 1, c)
}

func (*Imager) TestWrongFileType(c *C) {
	data := readSample("sample.txt", c)
	code, err := newThumbnail(data, types.Image{}, nil)
	c.Assert(err, ErrorMatches, "unsupported file type.*")
	c.Assert(code, Equals, 400)
}

func (*Imager) TestNewThumbnail(c *C) {
	const (
		id   = "123"
		name = "sample.jpg"
	)
	data := readSample(name, c)
	img := types.Image{
		ImageCommon: types.ImageCommon{
			SHA1: id,
		},
		Imgnm: name,
	}

	std := img
	std.FileType = jpeg
	std.Dims = jpegDims
	std.Imgnm = name
	std.MD5 = "60e41092581f7b329b057b8402caa8a7"
	std.Size = 300792

	var wg sync.WaitGroup
	wg.Add(1)
	ch := make(chan types.Image)
	go func() {
		defer wg.Done()
		c.Assert(<-ch, Equals, std)
	}()

	code, err := newThumbnail(data, img, &websockets.Client{
		AllocateImage: ch,
	})
	c.Assert(err, IsNil)
	c.Assert(code, Equals, 200)
	wg.Wait()

	assertDBRecord(std, c)
	assertFiles(name, id, jpeg, c)
}

// Assert the image record in the database matches the sample
func assertDBRecord(img types.Image, c *C) {
	c.Assert(getImageRecord(img.SHA1, c), Equals, img.ImageCommon)
}

func getImageRecord(id string, c *C) (res types.ImageCommon) {
	c.Assert(db.DB(db.GetImage(id)).One(&res), IsNil)
	return
}

// Assert image file assets were created with the correct paths
func assertFiles(src, id string, fileType uint8, c *C) {
	var (
		paths [3]string
		data  [3][]byte
	)
	paths[0] = filepath.FromSlash("test/" + src)
	destPaths := getFilePaths(id, fileType)
	paths[1], paths[2] = destPaths[0], destPaths[1]

	for i := range paths {
		var err error
		data[i], err = ioutil.ReadFile(paths[i])
		c.Assert(err, IsNil)
	}

	c.Assert(data[0], DeepEquals, data[1])
	c.Assert(len(data[1]) > len(data[2]), Equals, true)
}

func (*Imager) TestAPNGThumbnailing(c *C) {
	for _, ext := range [...]string{"png", "apng"} {
		img := types.Image{
			ImageCommon: types.ImageCommon{
				SHA1: ext,
			},
		}
		data := readSample("sample."+ext, c)

		_, err := newThumbnail(data, img, newClient())
		c.Assert(err, IsNil)

		c.Assert(getImageRecord(ext, c).APNG, Equals, ext == "apng")
	}
}

func newClient() *websockets.Client {
	ch := make(chan types.Image)
	go func() {
		<-ch
	}()
	return &websockets.Client{
		AllocateImage: ch,
	}
}

func (*Imager) TestNonExistantClient(c *C) {
	b, w := newMultiWriter()
	writeFields(c, w, map[string]string{
		"id": "123",
	})
	req := newRequest(c, b, w)
	req.Header.Set("Content-Length", "300792")

	code, err := newImageUpload(req)
	c.Assert(err, ErrorMatches, "no client found: 123")
	c.Assert(code, Equals, 400)
}

func (*Imager) TestNoImageUploaded(c *C) {
	b, w := newMultiWriter()
	writeFields(c, w, syncClient())
	req := newRequest(c, b, w)
	req.Header.Set("Content-Length", "300792")

	code, err := newImageUpload(req)
	c.Assert(err, ErrorMatches, "http: no such file")
	c.Assert(code, Equals, 400)
}

func (*Imager) TestThumbNailReuse(c *C) {
	for i := 0; i < 2; i++ {
		req, wg := newJPEGRequest(c)

		code, err := newImageUpload(req)
		c.Assert(err, IsNil)
		c.Assert(code, Equals, 200)

		wg.Wait()
	}
}

func newJPEGRequest(c *C) (*http.Request, *sync.WaitGroup) {
	var wg sync.WaitGroup
	wg.Add(1)
	b, w := newMultiWriter()
	fields := assertImage(stdJPEG, &wg, c)
	fields["spoiler"] = "1"
	writeFields(c, w, fields)

	file, err := w.CreateFormFile("image", stdJPEG.Imgnm)
	c.Assert(err, IsNil)
	_, err = file.Write(readSample(stdJPEG.Imgnm, c))
	c.Assert(err, IsNil)

	req := newRequest(c, b, w)
	req.Header.Set("Content-Length", "300792")

	return req, &wg
}

// Assert the image sent to the client matches the standard
func assertImage(std types.Image, wg *sync.WaitGroup, c *C) map[string]string {
	ch := make(chan types.Image)
	cl := &websockets.Client{
		AllocateImage: ch,
	}
	websockets.Clients.Add(cl, "1")

	go func() {
		defer wg.Done()
		c.Assert(<-ch, Equals, std)
	}()

	return map[string]string{"id": cl.ID}
}

func (*Imager) TestUploadHandler(c *C) {
	req, wg := newJPEGRequest(c)
	rec := httptest.NewRecorder()

	NewImageUpload(rec, req)
	acao := rec.Header().Get("Access-Control-Allow-Origin")
	c.Assert(acao, Equals, config.Get().HTTP.Origin)
	c.Assert(rec.Code, Equals, 200)
	wg.Wait()
}

func (*Imager) TestErrorPassing(c *C) {
	const ip = "::1"
	req, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, IsNil)
	req.RemoteAddr = ip
	rec := httptest.NewRecorder()

	NewImageUpload(rec, req)
	c.Assert(rec.Code, Equals, 400)
	const errMsg = "strconv.ParseInt: parsing \"\": invalid syntax\n"
	c.Assert(rec.Body.String(), Equals, errMsg)
}
