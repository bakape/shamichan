package imager

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager/assets"
	. "github.com/bakape/meguca/test"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
)

func newMultiWriter() (*bytes.Buffer, *multipart.Writer) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	return body, writer
}

func newRequest(
	t *testing.T,
	body io.Reader,
	w *multipart.Writer,
) *http.Request {
	req := httptest.NewRequest("PUT", "/", body)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func setHeaders(req *http.Request, headers map[string]string) {
	for key, val := range headers {
		req.Header.Set(key, val)
	}
}

func assertCode(t *testing.T, res, std int) {
	if res != std {
		t.Errorf("unexpected status code: %d : %d", std, res)
	}
}

func assertTableClear(t *testing.T, tables ...string) {
	if err := db.ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}

func assertInsert(t *testing.T, table string, doc interface{}) {
	if err := db.Insert(table, doc); err != nil {
		t.Fatal(err)
	}
}

func newJPEGRequest(t *testing.T) *http.Request {
	var wg sync.WaitGroup
	wg.Add(1)
	b, w := newMultiWriter()

	file, err := w.CreateFormFile("image", assets.StdJPEG.Name)
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.Write(readSample(t, assets.StdJPEG.Name))
	if err != nil {
		t.Fatal(err)
	}

	req := newRequest(t, b, w)
	req.Header.Set("Content-Length", "300792")

	return req
}

func getImageRecord(t *testing.T, id string) (res types.ImageCommon) {
	if err := db.One(db.GetImage(id), &res); err != nil {
		t.Fatal(err)
	}
	return
}

// Assert image file assets were created with the correct paths
func assertFiles(t *testing.T, src, id string, fileType uint8) {
	var (
		paths [3]string
		data  [3][]byte
	)
	paths[0] = filepath.Join("testdata", src)
	destPaths := assets.GetFilePaths(id, fileType)
	paths[1], paths[2] = destPaths[0], destPaths[1]

	for i := range paths {
		var err error
		data[i], err = ioutil.ReadFile(paths[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	AssertBufferEquals(t, data[0], data[1])
	if len(data[1]) < len(data[2]) {
		t.Error("unexpected file size diffrence")
	}
}

func assertImageRefCount(t *testing.T, id string, count int) {
	var posts int
	if err := db.One(db.GetImage(id).Field("posts"), &posts); err != nil {
		t.Fatal(err)
	}
	if posts != count {
		t.Errorf("unexpected post count: %d : %d", count, posts)
	}
}

func assertImageToken(t *testing.T, id, SHA1, name string) {
	q := r.Table("imageTokens").Get(id).Field("SHA1").Eq(SHA1)
	var isEqual bool
	if err := db.One(q, &isEqual); err != nil {
		t.Fatal(err)
	}
	if !isEqual {
		t.Error("SHA1 hash mismatch")
	}
}

func TestDetectFileType(t *testing.T) {
	t.Parallel()

	// Supported file types
	for c, e := range types.Extensions {
		code := c
		ext := e
		t.Run(ext, func(t *testing.T) {
			t.Parallel()

			typ, err := detectFileType(readSample(t, "sample."+ext))
			if err != nil {
				t.Fatal(err)
			}
			if typ != code {
				t.Fatalf("unexpected type code: %d : %d", code, typ)
			}
		})
	}
}

func TestInvalidContentLengthHeader(t *testing.T) {
	b, w := newMultiWriter()
	req := newRequest(t, b, w)
	setHeaders(req, map[string]string{
		"Content-Length": "KAWFEE",
	})

	err := parseUploadForm(req)
	if s := fmt.Sprint(err); !strings.Contains(s, "invalid syntax") {
		UnexpectedError(t, err)
	}
}

func TestUploadTooLarge(t *testing.T) {
	conf := config.Get()
	(*conf).MaxSize = 1
	b, w := newMultiWriter()
	req := newRequest(t, b, w)
	req.Header.Set("Content-Length", "1048577")

	if err := parseUploadForm(req); err != errTooLarge {
		UnexpectedError(t, err)
	}
}

func TestInvalidForm(t *testing.T) {
	b, w := newMultiWriter()
	req := newRequest(t, b, w)
	setHeaders(req, map[string]string{
		"Content-Length": "1024",
		"Content-Type":   "GWEEN TEA",
	})

	if parseUploadForm(req) == nil {
		t.Fatal("expected an error")
	}
}

func TestSuccessfulFormParse(t *testing.T) {
	b, w := newMultiWriter()
	req := newRequest(t, b, w)
	req.Header.Set("Content-Length", "1024")

	if err := parseUploadForm(req); err != nil {
		t.Fatal(err)
	}
}

func TestWrongFileType(t *testing.T) {
	data := readSample(t, "sample.txt")

	code, _, err := newThumbnail(data, types.ImageCommon{})

	if s := fmt.Sprint(err); !strings.HasPrefix(s, "unsupported file type") {
		UnexpectedError(t, err)
	}
	assertCode(t, code, 400)
}

func TestNewThumbnail(t *testing.T) {
	assertTableClear(t, "images", "imageTokens")
	resetDirs(t)

	req := newJPEGRequest(t)
	rec := httptest.NewRecorder()
	NewImageUpload(rec, req)
	assertCode(t, rec.Code, 200)

	std := types.ProtoImage{
		ImageCommon: assets.StdJPEG.ImageCommon,
		Posts:       1,
	}
	var img types.ProtoImage
	if err := db.One(r.Table("images").Get(std.SHA1), &img); err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, img, std)

	assertImageToken(t, rec.Body.String(), std.SHA1, assets.StdJPEG.Name)
	assertFiles(t, "sample.jpg", std.SHA1, types.JPEG)
}

func TestAPNGThumbnailing(t *testing.T) {
	assertTableClear(t, "images", "imageTokens")
	resetDirs(t)

	for _, e := range [...]string{"png", "apng"} {
		ext := e
		t.Run(ext, func(t *testing.T) {
			t.Parallel()

			img := types.ImageCommon{
				SHA1: ext,
			}
			data := readSample(t, "sample."+ext)

			if _, _, err := newThumbnail(data, img); err != nil {
				t.Fatal(err)
			}

			if getImageRecord(t, ext).APNG != (ext == "apng") {
				t.Fatal("unexpected APNG flag value")
			}
		})
	}
}

func TestNoImageUploaded(t *testing.T) {
	b, w := newMultiWriter()
	req := newRequest(t, b, w)
	req.Header.Set("Content-Length", "300792")

	code, _, err := newImageUpload(req)
	if err != http.ErrMissingFile {
		UnexpectedError(t, err)
	}
	assertCode(t, code, 400)
}

func TestThumbNailReuse(t *testing.T) {
	assertTableClear(t, "images", "imageTokens")
	resetDirs(t)

	for i := 1; i <= 2; i++ {
		req := newJPEGRequest(t)
		code, _, err := newImageUpload(req)
		if err != nil {
			t.Fatal(err)
		}
		assertCode(t, code, 200)

		assertImageRefCount(t, assets.StdJPEG.SHA1, i)
	}
}

func TestErrorPassing(t *testing.T) {
	t.Parallel()

	const ip = "::1"
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = ip
	rec := httptest.NewRecorder()

	NewImageUpload(rec, req)

	assertCode(t, rec.Code, 400)
	const errMsg = "strconv.ParseInt: parsing \"\": invalid syntax\n"
	if s := rec.Body.String(); s != errMsg {
		t.Errorf("unexpected body: `%s`", s)
	}
}
