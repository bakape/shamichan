package server

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/templates"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	"github.com/dimfeld/httptreemux"
	. "gopkg.in/check.v1"
)

const (
	notFound = "404 Not found"
)

func Test(t *testing.T) { TestingT(t) }

var genericImage = &types.Image{
	ImageCommon: types.ImageCommon{
		SHA1: "foo",
	},
}

// Does not seem like we can easily resuse testing functions. Thus copy/paste
// for now.
type DB struct {
	r http.Handler
}

var testDBName string

var _ = Suite(&DB{})

func (d *DB) SetUpSuite(c *C) {
	db.DBName = db.UniqueDBName()
	c.Assert(db.Connect(), IsNil)
	c.Assert(db.InitDB(), IsNil)
	setupPosts(c)
	config.Set(config.Configs{})
	d.r = createRouter()
}

func (*DB) SetUpTest(_ *C) {
	config.Set(config.Configs{
		Boards: []string{"a"},
	})
}

func (d *DB) TearDownSuite(c *C) {
	c.Assert(r.DBDrop(db.DBName).Exec(db.RSession), IsNil)
	c.Assert(db.RSession.Close(), IsNil)
}

// Create a multipurpose set of threads and posts for tests
func setupPosts(c *C) {
	threads := []types.DatabaseThread{
		{
			ID:    1,
			Board: "a",
			Log:   dummyLog(11),
			Posts: map[string]types.Post{
				"1": {
					ID:    1,
					OP:    1,
					Board: "a",
					Image: genericImage,
				},
				"2": {
					ID:    2,
					OP:    1,
					Board: "a",
				},
			},
		},
		{
			ID:    3,
			Board: "a",
			Log:   dummyLog(33),
			Posts: map[string]types.Post{
				"3": {
					ID:    3,
					Image: genericImage,
					OP:    3,
					Board: "a",
				},
			},
		},
		{
			ID:    4,
			Board: "c",
			Log:   dummyLog(44),
			Posts: map[string]types.Post{
				"4": {
					ID:    4,
					Image: genericImage,
					OP:    4,
					Board: "c",
				},
			},
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(threads)), IsNil)

	infoUpdate := db.GetMain("info").Update(map[string]int{"postCtr": 8})
	histUpdate := db.GetMain("histCounts").Update(map[string]int{"a": 7})
	c.Assert(db.Write(infoUpdate), IsNil)
	c.Assert(db.Write(histUpdate), IsNil)
}

func dummyLog(n int) [][]byte {
	log := make([][]byte, n)
	for i := 0; i < n; i++ {
		log[i] = []byte{1}
	}
	return log
}

type WebServer struct {
	r http.Handler
}

var _ = Suite(&WebServer{})

func (w *WebServer) SetUpSuite(c *C) {
	webRoot = "test"
	w.r = createRouter()
}

func (*WebServer) SetUpTest(_ *C) {
	config.Set(config.Configs{
		Boards: []string{"a", "c"},
	})
	config.SetClient(nil, "")
}

func (w *WebServer) TestFrontpageRedirect(c *C) {
	config.Set(config.Configs{
		Frontpage: filepath.FromSlash("test/frontpage.html"),
	})
	req := newRequest(c, "/")
	rec := httptest.NewRecorder()
	w.r.ServeHTTP(rec, req)
	assertBody(rec, "<!doctype html><html></html>\n", c)
	assertCode(rec, 200, c)
}

func (w *WebServer) TestAllBoardRedirect(c *C) {
	rec := httptest.NewRecorder()
	req := newRequest(c, "/")
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 302, c)
	c.Assert(rec.Header().Get("Location"), Equals, "/all/")
}

func assertEtag(rec *httptest.ResponseRecorder, etag string, c *C) {
	c.Assert(rec.Header().Get("ETag"), Equals, etag)
}

func assertBody(rec *httptest.ResponseRecorder, body string, c *C) {
	c.Assert(rec.Body.String(), DeepEquals, body)
}

func assertCode(rec *httptest.ResponseRecorder, status int, c *C) {
	c.Assert(rec.Code, Equals, status)
}

func assertHeaders(c *C, rec *httptest.ResponseRecorder, h map[string]string) {
	for key, val := range h {
		c.Assert(rec.Header().Get(key), Equals, val)
	}
}

func newRequest(c *C, url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	c.Assert(err, IsNil)
	return req
}

func newPair(c *C, url string) (*httptest.ResponseRecorder, *http.Request) {
	return httptest.NewRecorder(), newRequest(c, url)
}

func (w *WebServer) TestConfigServing(c *C) {
	etag := "foo"
	config.SetClient([]byte{1}, etag)

	rec, req := newPair(c, "/json/config")
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 200, c)
	assertBody(rec, string([]byte{1}), c)
	assertEtag(rec, etag, c)

	// And with etag
	rec, req = newPair(c, "/json/config")
	req.Header.Set("If-None-Match", etag)
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (w *WebServer) TestText404(c *C) {
	rec, req := newPair(c, "/lalala/")
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)
	assertBody(rec, notFound, c)
}

func (w *WebServer) TestPanicHandler(c *C) {
	webRoot = "test"
	r := httptreemux.New()
	h := wrapHandler(func(_ http.ResponseWriter, _ *http.Request) {
		panic(errors.New("foo"))
	})
	r.GET("/panic", h)
	r.PanicHandler = textErrorPage
	rec := httptest.NewRecorder()
	req := newRequest(c, "/panic")

	// Prevent printing stack trace to terminal
	log.SetOutput(ioutil.Discard)
	r.ServeHTTP(rec, req)
	log.SetOutput(os.Stdout)

	assertCode(rec, 500, c)
	assertBody(rec, "500 foo", c)
}

func (w *WebServer) TestText500(c *C) {
	rec, req := newPair(c, "/")
	text404(rec, req)
	assertCode(rec, 404, c)
	assertBody(rec, "404 Not found", c)
}

func (*WebServer) TestSetHeaders(c *C) {
	// HTML
	rec := httptest.NewRecorder()
	const etag = "foo"
	headers := map[string]string{
		"X-Frame-Options": "sameorigin",
		"Cache-Control":   "max-age=0, must-revalidate",
		"Expires":         "Fri, 01 Jan 1990 00:00:00 GMT",
		"ETag":            etag,
	}
	setHeaders(rec, etag)
	assertHeaders(c, rec, headers)
}

func (*WebServer) TestDetectLastN(c *C) {
	// No lastN query string
	req := newRequest(c, "/a/1")
	c.Assert(detectLastN(req), Equals, 0)

	// ?lastN value within bounds
	req = newRequest(c, "/a/1?lastN=100")
	c.Assert(detectLastN(req), Equals, 100)

	// ?lastN value beyond max
	req = newRequest(c, "/a/1?lastN=1000")
	c.Assert(detectLastN(req), Equals, 0)
}

func (*WebServer) TestImageServer(c *C) {
	const (
		truncated         = "/src/tis life.gif"
		notFoundTruncated = "src/nobody here.gif"
	)
	imageWebRoot = "test"
	path := imageWebRoot + truncated
	notFound := imageWebRoot + notFoundTruncated

	// Succesful first serve
	rec, req := newPair(c, path)
	params := map[string]string{
		"path": truncated,
	}
	serveImages(rec, req, params)
	buf, err := ioutil.ReadFile(path)
	c.Assert(err, IsNil)
	assertBody(rec, string(buf), c)
	headers := map[string]string{
		"Cache-Control":   "max-age=30240000",
		"X-Frame-Options": "sameorigin",
		"ETag":            "0",
	}
	assertHeaders(c, rec, headers)

	// Fake etag validation
	rec, req = newPair(c, path)
	req.Header.Set("If-None-Match", "0")
	serveImages(rec, req, params)
	assertCode(rec, 304, c)

	// Non-existing file
	rec, req = newPair(c, notFound)
	params["path"] = notFoundTruncated
	serveImages(rec, req, params)
	assertCode(rec, 404, c)
}

func (*WebServer) TestCompareEtag(c *C) {
	// Etag comparison
	rec, req := newPair(c, "/")
	const etag = "foo"
	req.Header.Set("If-None-Match", etag)
	c.Assert(pageEtag(rec, req, etag), Equals, false)

	rec, req = newPair(c, "")
	headers := map[string]string{
		"ETag":          etag,
		"Cache-Control": "max-age=0, must-revalidate",
	}
	pageEtag(rec, req, etag)
	assertHeaders(c, rec, headers)
}

func (*WebServer) TestEtagStart(c *C) {
	c.Assert(etagStart(1), Equals, "W/1")
}

func (w *WebServer) TestServeIndexTemplate(c *C) {
	const (
		desktopUA = "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 " +
			"(KHTML, like Gecko) Chrome/41.0.2228.0 Safari/537.36"
		mobileUA = "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus" +
			" Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko)" +
			" Chrome/18.0.1025.166 Mobile Safari/535.19"
	)
	desktop := templates.Store{
		HTML: []byte("desktop"),
		Hash: "dhash",
	}
	mobile := templates.Store{
		HTML: []byte("mobile"),
		Hash: "mhash",
	}
	templates.Set("index", desktop)
	templates.Set("mobile", mobile)
	headers := map[string]string{
		"Content-Type": "text/html",
	}

	// Desktop
	rec, req := newPair(c, "/a/")
	req.Header.Set("User-Agent", desktopUA)
	w.r.ServeHTTP(rec, req)
	assertBody(rec, string(desktop.HTML), c)
	assertEtag(rec, desktop.Hash, c)
	assertHeaders(c, rec, headers)

	// Mobile
	rec, req = newPair(c, "/a/")
	req.Header.Set("User-Agent", mobileUA)
	w.r.ServeHTTP(rec, req)
	assertBody(rec, string(mobile.HTML), c)
	assertEtag(rec, mobile.Hash+"-mobile", c)
	assertHeaders(c, rec, headers)

	// Etag matches
	rec, req = newPair(c, "/a/")
	req.Header.Set("If-None-Match", desktop.Hash)
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func removeIndentation(s string) string {
	s = strings.Replace(s, "\t", "", -1)
	s = strings.Replace(s, "\n", "", -1)
	return s
}

func (d *DB) TestThreadHTML(c *C) {
	body := []byte("body")
	templates.Set("index", templates.Store{
		HTML: body,
		Hash: "hash",
	})
	webRoot = "test"

	// Unparsable thread number
	rec, req := newPair(c, "/a/www")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Non-existant thread
	rec, req = newPair(c, "/a/22")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Thread exists
	rec, req = newPair(c, "/a/1")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, string(body), c)
}

func (d *DB) TestServePost(c *C) {
	// Invalid post number
	rec, req := newPair(c, "/json/post/www")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Non-existing post or otherwise invalid post
	rec, req = newPair(c, "/json/post/66")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Existing post
	const (
		etag = "d96fa6542aaf4c9e"
		body = `{"editing":false,"op":1,"id":2,"time":0,"board":"a","body":""}`
	)
	rec, req = newPair(c, "/json/post/2")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, body, c)
	assertEtag(rec, etag, c)

	// Etags match
	rec, req = newPair(c, "/json/post/2")
	req.Header.Set("If-None-Match", etag)
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (d *DB) TestBoardJSON(c *C) {
	// Invalid board
	rec, req := newPair(c, "/json/nope/")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	rec, req = newPair(c, "/json/a/")
	const body = `
{
	"ctr":7,
	"threads":[
		{
			"postCtr":0,
			"imageCtr":0,
			"editing":false,
			"id":3,
			"time":0,
			"board":"a",
			"body":"",
			"image":{
				"fileType":0,
				"dims":[0,0,0,0],
				"size":0,
				"MD5":"",
				"SHA1":"foo",
				"imgnm":""
			},
			"logCtr":33,
			"bumpTime":0,
			"replyTime":0
		},
		{
			"postCtr":0,
			"imageCtr":0,
			"editing":false,
			"id":1,
			"time":0,
			"board":"a",
			"body":"",
			"image":{
				"fileType":0,
				"dims":[0,0,0,0],
				"size":0,
				"MD5":"",
				"SHA1":"foo",
				"imgnm":""
			},
			"logCtr":11,
			"bumpTime":0,
			"replyTime":0
		}
	]
}`
	d.r.ServeHTTP(rec, req)
	assertBody(rec, removeIndentation(body), c)
	const etag = "W/7"
	assertEtag(rec, etag, c)

	// Etags match
	rec, req = newPair(c, "/json/a/")
	req.Header.Set("If-None-Match", etag)
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (d *DB) TestAllBoardJSON(c *C) {
	const etag = "W/8"
	const body = `
{
	"ctr":8,
	"threads":[
		{
			"postCtr":0,
			"imageCtr":0,
			"editing":false,
			"id":4,
			"time":0,
			"board":"c",
			"body":"",
			"image":{
				"fileType":0,
				"dims":[0,0,0,0],
				"size":0,
				"MD5":"",
				"SHA1":"foo",
				"imgnm":""
			},
			"logCtr":44,
			"bumpTime":0,
			"replyTime":0
		},
		{
			"postCtr":0,
			"imageCtr":0,
			"editing":false,
			"id":3,
			"time":0,
			"board":"a",
			"body":"",
			"image":{
				"fileType":0,
				"dims":[0,0,0,0],
				"size":0,
				"MD5":"",
				"SHA1":"foo",
				"imgnm":""
			},
			"logCtr":33,
			"bumpTime":0,
			"replyTime":0
		},
		{
			"postCtr":0,
			"imageCtr":0,
			"editing":false,
			"id":1,
			"time":0,
			"board":"a",
			"body":"",
			"image":{
				"fileType":0,
				"dims":[0,0,0,0],
				"size":0,
				"MD5":"",
				"SHA1":"foo",
				"imgnm":""
			},
			"logCtr":11,
			"bumpTime":0,
			"replyTime":0
		}
	]
}`
	rec, req := newPair(c, "/json/all/")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, removeIndentation(body), c)
	assertEtag(rec, etag, c)

	// Etags match
	rec, req = newPair(c, "/json/all/")
	req.Header.Set("If-None-Match", etag)
	d.r.ServeHTTP(rec, req)
	allBoardJSON(rec, req)
	assertCode(rec, 304, c)
}

func (d *DB) TestThreadJSON(c *C) {
	// Invalid board
	rec, req := newPair(c, "/json/nope/1")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Invalid post number
	rec, req = newPair(c, "/json/a/www")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Non-existing thread
	rec, req = newPair(c, "/json/a/22")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Valid thread request
	const body = `
{
	"postCtr":0,
	"imageCtr":0,
	"editing":false,
	"id":1,
	"time":0,
	"board":"a",
	"body":"",
	"image":{
		"fileType":0,
		"dims":[0,0,0,0],
		"size":0,
		"MD5":"",
		"SHA1":"foo",
		"imgnm":""
	},
	"logCtr":11,
	"bumpTime":0,
	"replyTime":0,
	"posts":{
		"2":{
			"editing":false,
			"op":1,
			"id":2,
			"time":0,
			"board":"a",
			"body":""
		}
	}
}`
	const etag = "W/11"
	rec, req = newPair(c, "/json/a/1")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, removeIndentation(body), c)
	assertEtag(rec, etag, c)

	// Etags match
	rec, req = newPair(c, "/json/a/1")
	req.Header.Set("If-None-Match", etag)
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (w *WebServer) TestGzip(c *C) {
	config.Set(config.Configs{
		Gzip: true,
	})
	r := createRouter()
	rec, req := newPair(c, "/json/config")
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(rec, req)
	c.Assert(rec.Header().Get("Content-Encoding"), Equals, "gzip")
}

func (w *WebServer) TestProxyHeaders(c *C) {
	const ip = "68.180.194.242"
	config.Set(config.Configs{
		TrustProxies: true,
	})
	r := createRouter()
	rec, req := newPair(c, "/json/config")
	req.Header.Set("X-Forwarded-For", ip)
	req.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(rec, req)
	c.Assert(req.RemoteAddr, Equals, ip+":1234")
}

func (w *WebServer) TestAssetServer(c *C) {
	rec, req := newPair(c, "/assets/frontpage.html")
	w.r.ServeHTTP(rec, req)
	assertBody(rec, "<!doctype html><html></html>\n", c)
}

func (w *WebServer) TestServeWorker(c *C) {
	rec, req := newPair(c, "/worker.js")
	w.r.ServeHTTP(rec, req)
	assertBody(rec, "console.log(\"Worker dess\")\n", c)
}
