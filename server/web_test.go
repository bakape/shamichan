package server

import (
	"errors"
	"github.com/julienschmidt/httprouter"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
)

type WebServer struct{}

var _ = Suite(&WebServer{})

func (*WebServer) TestFrontpageRedirect(c *C) {
	config = serverConfigs{}
	config.Frontpage = "./test/frontpage.html"
	server := httptest.NewServer(http.HandlerFunc(redirectToDefault))
	defer server.Close()
	res, err := http.Get(server.URL)
	c.Assert(err, IsNil)
	frontpage, err := ioutil.ReadAll(res.Body)
	c.Assert(err, IsNil)
	c.Assert(res.Body.Close(), IsNil)
	c.Assert(string(frontpage), Equals, "<!doctype html><html></html>\n")
}

func (*WebServer) TestDefaultBoardRedirect(c *C) {
	config = serverConfigs{}
	config.Boards.Default = "a"
	rec := runHandler(c, redirectToDefault)
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

func runHandler(c *C, h http.HandlerFunc) *httptest.ResponseRecorder {
	req := newRequest(c)
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func newRequest(c *C) *http.Request {
	req, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, IsNil)
	return req
}

func (*WebServer) TestConfigServing(c *C) {
	configHash = "foo"
	clientConfig = clientConfigs{}
	etag := "W/" + configHash
	rec := runHandler(c, serveConfigs)
	assertCode(rec, 200, c)
	assertBody(rec, string(marshalJSON(clientConfig)), c)
	assertEtag(rec, etag, c)

	// And with etag
	rec = httptest.NewRecorder()
	req := newRequest(c)
	req.Header.Set("If-None-Match", etag)
	serveConfigs(rec, req)
	assertCode(rec, 304, c)
}

func (*WebServer) TestEtagComparison(c *C) {
	req := newRequest(c)
	const etag = "foo"
	req.Header.Set("If-None-Match", etag)
	rec := httptest.NewRecorder()
	c.Assert(checkClientEtag(rec, req, etag), Equals, true)
}

func (*WebServer) TestNotFoundHandler(c *C) {
	webRoot = "./test"
	rec := runHandler(c, notFoundHandler)
	assertBody(rec, "<!doctype html><html>404</html>\n", c)
	assertCode(rec, 404, c)
	headers := map[string]string{
		"Content-Type":           "text/html; charset=UTF-8",
		"X-Content-Type-Options": "nosniff",
	}
	assertHeaders(c, rec, headers)
}

func (*WebServer) TestText404(c *C) {
	rec := runHandler(c, func(res http.ResponseWriter, _ *http.Request) {
		text404(res)
	})
	assertCode(rec, 404, c)
	assertBody(rec, "404 Not found\n", c)
}

func (*WebServer) TestPanicHandler(c *C) {
	webRoot = "./test"
	err := errors.New("foo")

	// Prevent printing stack trace to terminal
	log.SetOutput(ioutil.Discard)
	rec := runHandler(c, func(res http.ResponseWriter, req *http.Request) {
		panicHandler(res, req, err)
	})
	log.SetOutput(os.Stdout)
	assertCode(rec, 500, c)
	assertBody(rec, "<!doctype html><html>50x</html>\n", c)
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
		"Content-Type":    "text/html; charset=UTF-8",
	}
	setHeaders(rec, etag, false)
	assertHeaders(c, rec, headers)

	// JSON
	headers["Content-Type"] = "application/json; charset=UTF-8"
	rec = httptest.NewRecorder()
	setHeaders(rec, etag, true)
	assertHeaders(c, rec, headers)
}

func (*WebServer) TestDetectLastN(c *C) {
	// No lastN query string
	req := customRequest(c, "/a/1")
	c.Assert(detectLastN(req), Equals, 0)

	// ?lastN value within bounds
	req = customRequest(c, "/a/1?lastN=100")
	c.Assert(detectLastN(req), Equals, 100)

	// ?lastN value beyond max
	req = customRequest(c, "/a/1?lastN=1000")
	c.Assert(detectLastN(req), Equals, 0)
}

func customRequest(c *C, url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	c.Assert(err, IsNil)
	return req
}

func (*WebServer) TestImageServer(c *C) {
	const (
		truncated         = "/src/tis life.gif"
		notFoundTruncated = "src/nobody here.gif"
	)
	imageWebRoot = "./test"
	path := imageWebRoot + truncated
	notFound := imageWebRoot + notFoundTruncated

	// Succesful first serve
	req := customRequest(c, path)
	rec := httptest.NewRecorder()
	params := httprouter.Params{
		httprouter.Param{
			Key:   "filepath",
			Value: truncated,
		},
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
	req = customRequest(c, path)
	rec = httptest.NewRecorder()
	req.Header.Set("If-None-Match", "0")
	serveImages(rec, req, params)
	assertCode(rec, 304, c)

	// Non-existing file
	req = customRequest(c, notFound)
	rec = httptest.NewRecorder()
	params[0].Value = notFoundTruncated
	serveImages(rec, req, params)
	assertCode(rec, 404, c)
}

func (*WebServer) TestCompareEtag(c *C) {
	// Etag comparison
	rec := httptest.NewRecorder()
	req := newRequest(c)
	const etag = "foo"
	req.Header.Set("If-None-Match", etag)
	c.Assert(compareEtag(rec, req, etag, false), Equals, false)

	rec = httptest.NewRecorder()
	req = newRequest(c)
	headers := map[string]string{
		"ETag":          etag,
		"Content-Type":  "text/html; charset=UTF-8",
		"Cache-Control": "max-age=0, must-revalidate",
	}
	compareEtag(rec, req, etag, false)
	assertHeaders(c, rec, headers)
}

func (*WebServer) TestEtagStart(c *C) {
	c.Assert(etagStart(1), Equals, "W/1")
}

func (*WebServer) TestServeIndexTemplate(c *C) {
	const (
		desktopUA = "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 " +
			"(KHTML, like Gecko) Chrome/41.0.2228.0 Safari/537.36"
		mobileUA = "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus" +
			" Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko)" +
			" Chrome/18.0.1025.166 Mobile Safari/535.19"
	)
	desktop := templateStore{[]byte("desktop"), "dhash"}
	mobile := templateStore{[]byte("mobile"), "mhash"}
	resources = templateMap{}
	resources["index"] = desktop
	resources["mobile"] = mobile

	// Desktop
	req := newRequest(c)
	req.Header.Set("User-Agent", desktopUA)
	rec := httptest.NewRecorder()
	serveIndexTemplate(rec, req)
	assertBody(rec, string(desktop.HTML), c)
	assertEtag(rec, desktop.Hash, c)

	// Mobile
	req = newRequest(c)
	req.Header.Set("User-Agent", mobileUA)
	rec = httptest.NewRecorder()
	serveIndexTemplate(rec, req)
	assertBody(rec, string(mobile.HTML), c)
	assertEtag(rec, mobile.Hash+"-mobile", c)

	// Etag matches
	req = newRequest(c)
	req.Header.Set("If-None-Match", desktop.Hash)
	rec = httptest.NewRecorder()
	serveIndexTemplate(rec, req)
	assertCode(rec, 304, c)
}

func (*DB) TestThreadHTML(c *C) {
	body := []byte("body")
	resources = templateMap{
		"index": templateStore{
			HTML: body,
			Hash: "hash",
		},
	}
	webRoot = "./test"

	// Unparsable thread number
	rec := httptest.NewRecorder()
	threadHTML("a")(rec, newRequest(c), httprouter.Params{{Value: "www"}})
	assertCode(rec, 404, c)

	// Non-existant thread
	rec = httptest.NewRecorder()
	threadHTML("a")(rec, newRequest(c), httprouter.Params{{Value: "22"}})
	assertCode(rec, 404, c)

	// Thread exists
	setupBoardAccess()
	setupPosts()
	rec = httptest.NewRecorder()
	threadHTML("a")(rec, newRequest(c), httprouter.Params{{Value: "1"}})
	assertBody(rec, string(body), c)
}

func (*DB) TestServePost(c *C) {
	setupBoardAccess()
	setupPosts()

	// Invalid post number
	rec := httptest.NewRecorder()
	servePost(rec, newRequest(c), httprouter.Params{{Value: "www"}})
	assertCode(rec, 404, c)

	// Non-existing post or otherwise invalid post
	rec = httptest.NewRecorder()
	servePost(rec, newRequest(c), httprouter.Params{{Value: "66"}})
	assertCode(rec, 404, c)

	// Existing post
	rec = httptest.NewRecorder()
	const etag = "d96fa6542aaf4c9e"
	servePost(rec, newRequest(c), httprouter.Params{{Value: "2"}})
	assertBody(
		rec,
		`{"editing":false,"op":1,"id":2,"time":0,"board":"a","body":""}`,
		c,
	)
	assertEtag(rec, etag, c)

	// Etags match
	rec = httptest.NewRecorder()
	req := newRequest(c)
	req.Header.Set("If-None-Match", etag)
	servePost(rec, req, httprouter.Params{{Value: "2"}})
	assertCode(rec, 304, c)
}

func (*DB) TestBoardJSON(c *C) {
	setupPosts()
	setupBoardAccess()

	rec := httptest.NewRecorder()
	boardJSON("a")(rec, newRequest(c))
	assertBody(
		rec,
		`{"ctr":7,"threads":[{"postCtr":0,"imageCtr":0,"bumpTime":0,`+
			`"replyTime":0,"editing":false,"src":"foo","time":0,"body":""},`+
			`{"postCtr":1,"imageCtr":0,"bumpTime":0,"replyTime":0,`+
			`"editing":false,"src":"foo","time":0,"body":""}]}`,
		c,
	)
	const etag = "W/7"
	assertEtag(rec, etag, c)

	// Etags match
	rec = httptest.NewRecorder()
	req := newRequest(c)
	req.Header.Set("If-None-Match", etag)
	boardJSON("a")(rec, req)
	assertCode(rec, 304, c)
}

func (*DB) TestAllBoardJSON(c *C) {
	setupBoardAccess()
	setupPosts()

	rec := httptest.NewRecorder()
	allBoardJSON(rec, newRequest(c))
	const etag = "W/8"
	assertBody(
		rec,
		`{"ctr":8,"threads":[{"postCtr":0,"imageCtr":0,"bumpTime":0,`+
			`"replyTime":0,"editing":false,"src":"foo","time":0,"body":""},`+
			`{"postCtr":0,"imageCtr":0,"bumpTime":0,"replyTime":0,`+
			`"editing":false,"src":"foo","time":0,"body":""},{"postCtr":1,`+
			`"imageCtr":0,"bumpTime":0,"replyTime":0,"editing":false,`+
			`"src":"foo","time":0,"body":""}]}`,
		c,
	)
	assertEtag(rec, etag, c)

	// Etags match
	rec = httptest.NewRecorder()
	req := newRequest(c)
	req.Header.Set("If-None-Match", etag)
	allBoardJSON(rec, req)
	assertCode(rec, 304, c)
}

func (*DB) TestThreadJSON(c *C) {
	setupBoardAccess()
	setupPosts()

	// Invalid post number
	rec := httptest.NewRecorder()
	threadJSON("a")(rec, newRequest(c), httprouter.Params{{Value: "www"}})
	assertCode(rec, 404, c)

	// Non-existing thread
	rec = httptest.NewRecorder()
	threadJSON("a")(rec, newRequest(c), httprouter.Params{{Value: "22"}})
	assertCode(rec, 404, c)

	// Valid thread request
	rec = httptest.NewRecorder()
	threadJSON("a")(rec, newRequest(c), httprouter.Params{{Value: "1"}})
	assertBody(
		rec,
		`{"postCtr":1,"imageCtr":0,"bumpTime":0,"replyTime":0,"editing":false,`+
			`"src":"foo","time":0,"body":"","posts":{"2":{"editing":false,`+
			`"op":1,"id":2,"time":0,"board":"a","body":""}}}`,
		c,
	)
	const etag = "W/1"
	assertEtag(rec, etag, c)

	// Etags match
	rec = httptest.NewRecorder()
	req := newRequest(c)
	req.Header.Set("If-None-Match", etag)
	threadJSON("a")(rec, req, httprouter.Params{{Value: "1"}})
	assertCode(rec, 304, c)
}

type routeCheck struct {
	path   string
	params httprouter.Params
}

func (*WebServer) TestCreateRouter(c *C) {
	config = serverConfigs{}
	config.Boards.Enabled = []string{"a"}
	r := createRouter()
	gets := [...]routeCheck{
		{"/", nil},
		{"/all/", nil},
		{"/api/all/", nil},
		{"/a/", nil},
		{"/api/a/", nil},
		{"/a/1", httprouter.Params{{"thread", "1"}}},
		{"/api/a/1", httprouter.Params{{"thread", "1"}}},
		{"/api/config", nil},
		{"/api/post/1", httprouter.Params{{"post", "1"}}},
		{"/ass/favicon.gif", httprouter.Params{{"filepath", "/favicon.gif"}}},
		{
			"/img/src/madotsuki.png",
			httprouter.Params{{"filepath", "/src/madotsuki.png"}},
		},
	}
	for _, rc := range gets {
		assertRoute("GET", rc, r, c)
	}
	assertRoute("POST", routeCheck{"/upload", nil}, r, c)
}

func assertRoute(method string, rc routeCheck, r *httprouter.Router, c *C) {
	handle, params, _ := r.Lookup(method, rc.path)
	c.Assert(params, DeepEquals, rc.params)
	c.Assert(handle, NotNil, Commentf("No handler on path '%s'", rc.path))
}

func (*WebServer) TestWrapRouter(c *C) {
	config = serverConfigs{}

	// Test GZIP
	r := httprouter.New()
	r.HandlerFunc("GET", "/", func(res http.ResponseWriter, _ *http.Request) {
		_, err := res.Write([]byte("Kyoani is shit"))
		c.Assert(err, IsNil)
	})
	rec := httptest.NewRecorder()
	req := customRequest(c, "/")
	req.Header.Set("Accept-Encoding", "gzip")
	wrapRouter(r).ServeHTTP(rec, req)
	c.Assert(rec.Header().Get("Content-Encoding"), Equals, "gzip")

	// Test honouring "X-Forwarded-For" headers
	config.HTTP.TrustProxies = true
	r = httprouter.New()
	var remoteIP string
	r.HandlerFunc("GET", "/", func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write([]byte("Kyoani is shit"))
		c.Assert(err, IsNil)
		remoteIP = req.RemoteAddr
	})
	rec = httptest.NewRecorder()
	req = customRequest(c, "/")
	const ip = "68.180.194.242"
	req.Header.Set("X-Forwarded-For", ip)
	wrapRouter(r).ServeHTTP(rec, req)
	c.Assert(remoteIP, Equals, ip)
}
