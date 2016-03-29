package server

import (
	"errors"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/templates"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	"github.com/dimfeld/httptreemux"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

var genericImage = types.Image{File: "foo"}

// Does not seem like we can easily resuse testing functions. Thus copy/paste
// for now.
type DB struct {
	dbName string
	r      http.Handler
}

var testDBName string

var _ = Suite(&DB{})

func (d *DB) SetUpSuite(c *C) {
	d.dbName = uniqueDBName()
	connectToRethinkDb(c)
	c.Assert(db.DB()(r.DBCreate(d.dbName)).Exec(), IsNil)
	db.RSession.Use(d.dbName)
	c.Assert(db.CreateTables(), IsNil)
	c.Assert(db.CreateIndeces(), IsNil)
	setupPosts(c)
	d.r = createRouter()
}

// Returns a unique datatabase name. Needed so multiple concurent `go test`
// don't clash in the same database.
func uniqueDBName() string {
	return "meguca_tests_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func connectToRethinkDb(c *C) {
	var err error
	db.RSession, err = r.Connect(r.ConnectOpts{
		Address: "localhost:28015",
	})
	c.Assert(err, IsNil)
}

func (*DB) SetUpTest(_ *C) {
	config.Config = config.Server{}
	config.Config.Boards.Enabled = []string{"a"}
}

func (d *DB) TearDownSuite(c *C) {
	c.Assert(r.DBDrop(d.dbName).Exec(db.RSession), IsNil)
	c.Assert(db.RSession.Close(), IsNil)
}

// Create a multipurpose set of threads and posts for tests
func setupPosts(c *C) {
	threads := []types.Thread{
		{ID: 1, Board: "a"},
		{ID: 3, Board: "a"},
		{ID: 4, Board: "c"},
	}
	c.Assert(db.DB()(r.Table("threads").Insert(threads)).Exec(), IsNil)

	posts := []types.Post{
		{
			ID:    1,
			OP:    1,
			Board: "a",
			Image: genericImage,
		},
		{
			ID:    2,
			OP:    1,
			Board: "a",
		},
		{
			ID:    3,
			OP:    3,
			Board: "a",
			Image: genericImage,
		},
		{
			ID:    4,
			OP:    4,
			Board: "c",
			Image: genericImage,
		},
	}
	c.Assert(db.DB()(r.Table("posts").Insert(posts)).Exec(), IsNil)

	main := []map[string]interface{}{
		{
			"id": "histCounts",
			"a":  7,
		},
		{
			"id":      "info",
			"postCtr": 8,
		},
	}
	c.Assert(db.DB()(r.Table("main").Insert(main)).Exec(), IsNil)
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
	config.Config = config.Server{}
	config.Config.Boards.Enabled = []string{"a"}
	config.Hash = ""
	config.ClientConfig = nil
}

func (w *WebServer) TestFrontpageRedirect(c *C) {
	config.Config.Frontpage = filepath.FromSlash("test/frontpage.html")
	req := newRequest(c, "/")
	rec := httptest.NewRecorder()
	w.r.ServeHTTP(rec, req)
	assertBody(rec, "<!doctype html><html></html>\n", c)
	assertCode(rec, 200, c)
}

func (w *WebServer) TestDefaultBoardRedirect(c *C) {
	config.Config.Boards.Default = "a"
	rec := httptest.NewRecorder()
	req := newRequest(c, "/")
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 302, c)
	c.Assert(rec.Header().Get("Location"), Equals, "/a/")
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
	config.Hash = "foo"
	config.ClientConfig = []byte{1}
	etag := config.Hash

	rec, req := newPair(c, "/api/config")
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 200, c)
	assertBody(rec, string([]byte{1}), c)
	assertEtag(rec, etag, c)

	// And with etag
	rec, req = newPair(c, "/api/config")
	req.Header.Set("If-None-Match", etag)
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (w *WebServer) TestNotFoundHandler(c *C) {
	webRoot = "test"
	rec, req := newPair(c, "/lalala/")
	w.r.ServeHTTP(rec, req)
	assertBody(rec, "<!doctype html><html>404</html>\n", c)
	assertCode(rec, 404, c)
}

func (w *WebServer) TestText404(c *C) {
	rec, req := newPair(c, "/api/post/nope")
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)
	assertBody(rec, "404 Not found", c)
}

func (w *WebServer) TestPanicHandler(c *C) {
	webRoot = "test"
	r := httptreemux.New()
	h := wrapHandler(func(_ http.ResponseWriter, _ *http.Request) {
		panic(errors.New("foo"))
	})
	r.GET("/panic", h)
	r.PanicHandler = panicHandler
	rec := httptest.NewRecorder()
	req := newRequest(c, "/panic")

	// Prevent printing stack trace to terminal
	log.SetOutput(ioutil.Discard)
	r.ServeHTTP(rec, req)
	log.SetOutput(os.Stdout)

	assertCode(rec, 500, c)
	assertBody(rec, "<!doctype html><html>50x</html>\n", c)
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
	templates.Resources = templates.Map{
		"index":  desktop,
		"mobile": mobile,
	}

	// Desktop
	rec, req := newPair(c, "/a/")
	req.Header.Set("User-Agent", desktopUA)
	w.r.ServeHTTP(rec, req)
	assertBody(rec, string(desktop.HTML), c)
	assertEtag(rec, desktop.Hash, c)

	// Mobile
	rec, req = newPair(c, "/a/")
	req.Header.Set("User-Agent", mobileUA)
	w.r.ServeHTTP(rec, req)
	assertBody(rec, string(mobile.HTML), c)
	assertEtag(rec, mobile.Hash+"-mobile", c)

	// Etag matches
	rec, req = newPair(c, "/a/")
	req.Header.Set("If-None-Match", desktop.Hash)
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (d *DB) TestThreadHTML(c *C) {
	body := []byte("body")
	templates.Resources = templates.Map{
		"index": templates.Store{
			HTML: body,
			Hash: "hash",
		},
	}
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
	rec, req := newPair(c, "/api/post/www")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Non-existing post or otherwise invalid post
	rec, req = newPair(c, "/api/post/66")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Existing post
	const etag = "d96fa6542aaf4c9e"
	rec, req = newPair(c, "/api/post/2")
	d.r.ServeHTTP(rec, req)
	assertBody(
		rec,
		`{"editing":false,"op":1,"id":2,"time":0,"board":"a","body":""}`,
		c,
	)
	assertEtag(rec, etag, c)

	// Etags match
	rec, req = newPair(c, "/api/post/2")
	req.Header.Set("If-None-Match", etag)
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (d *DB) TestBoardJSON(c *C) {
	// Invalid board
	rec, req := newPair(c, "/api/nope/")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	rec, req = newPair(c, "/api/a/")
	const body = `{"ctr":7,"threads":[{"postCtr":0,"imageCtr":0,"bumpTime":0,` +
		`"replyTime":0,"editing":false,"file":"foo","time":0,"body":""},` +
		`{"postCtr":1,"imageCtr":0,"bumpTime":0,"replyTime":0,` +
		`"editing":false,"file":"foo","time":0,"body":""}]}`
	d.r.ServeHTTP(rec, req)
	assertBody(rec, body, c)
	const etag = "W/7"
	assertEtag(rec, etag, c)

	// Etags match
	rec, req = newPair(c, "/api/a/")
	req.Header.Set("If-None-Match", etag)
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (d *DB) TestAllBoardJSON(c *C) {
	const etag = "W/8"
	const body = `{"ctr":8,"threads":[{"postCtr":0,"imageCtr":0,"bumpTime":0,` +
		`"replyTime":0,"editing":false,"file":"foo","time":0,"body":""},` +
		`{"postCtr":0,"imageCtr":0,"bumpTime":0,"replyTime":0,` +
		`"editing":false,"file":"foo","time":0,"body":""},{"postCtr":1,` +
		`"imageCtr":0,"bumpTime":0,"replyTime":0,"editing":false,` +
		`"file":"foo","time":0,"body":""}]}`
	rec, req := newPair(c, "/api/all/")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, body, c)
	assertEtag(rec, etag, c)

	// Etags match
	rec, req = newPair(c, "/api/all/")
	req.Header.Set("If-None-Match", etag)
	d.r.ServeHTTP(rec, req)
	allBoardJSON(rec, req)
	assertCode(rec, 304, c)
}

func (d *DB) TestThreadJSON(c *C) {
	// Invalid board
	rec, req := newPair(c, "/api/nope/1")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Invalid post number
	rec, req = newPair(c, "/api/a/www")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Non-existing thread
	rec, req = newPair(c, "/api/a/22")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Valid thread request
	const body = `{"postCtr":1,"imageCtr":0,"bumpTime":0,"replyTime":0,` +
		`"editing":false,"file":"foo","time":0,"body":"","posts":{"2":` +
		`{"editing":false,"op":1,"id":2,"time":0,"board":"a","body":""}}}`
	const etag = "W/1"
	rec, req = newPair(c, "/api/a/1")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, body, c)
	assertEtag(rec, etag, c)

	// Etags match
	rec, req = newPair(c, "/api/a/1")
	req.Header.Set("If-None-Match", etag)
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (w *WebServer) TestGzip(c *C) {
	config.Config.HTTP.Gzip = true
	r := createRouter()
	rec, req := newPair(c, "/api/config")
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(rec, req)
	c.Assert(rec.Header().Get("Content-Encoding"), Equals, "gzip")
}

func (w *WebServer) TestProxyHeaders(c *C) {
	const ip = "68.180.194.242"
	config.Config.HTTP.TrustProxies = true
	r := createRouter()
	rec, req := newPair(c, "/api/config")
	req.Header.Set("X-Forwarded-For", ip)
	req.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(rec, req)
	c.Assert(req.RemoteAddr, Equals, ip+":1234")
}

func (w *WebServer) TestAssetServer(c *C) {
	rec, req := newPair(c, "/ass/frontpage.html")
	w.r.ServeHTTP(rec, req)
	assertBody(rec, "<!doctype html><html></html>\n", c)
}
