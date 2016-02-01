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
	c.Assert(rec.Code, Equals, 302)
	c.Assert(rec.Header().Get("Location"), Equals, "/a/")
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
	c.Assert(rec.Code, Equals, 200)
	c.Assert(rec.Body.String(), Equals, string(marshalJSON(clientConfig)))
	c.Assert(rec.Header().Get("ETag"), Equals, etag)

	// And with etag
	rec = httptest.NewRecorder()
	req := newRequest(c)
	req.Header.Set("If-None-Match", etag)
	serveConfigs(rec, req)
	c.Assert(rec.Code, Equals, 304)
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
	c.Assert(
		rec.Body.String(),
		Equals,
		"<!doctype html><html>404</html>\n",
	)
	c.Assert(rec.Code, Equals, 404)
	headers := map[string]string{
		"Content-Type":           "text/html; charset=UTF-8",
		"X-Content-Type-Options": "nosniff",
	}
	assertHeaders(c, rec, headers)
}

func assertHeaders(c *C, rec *httptest.ResponseRecorder, h map[string]string) {
	for key, val := range h {
		c.Assert(rec.Header().Get(key), Equals, val)
	}
}

func (*WebServer) TestText404(c *C) {
	rec := runHandler(c, func(res http.ResponseWriter, _ *http.Request) {
		text404(res)
	})
	c.Assert(rec.Code, Equals, 404)
	c.Assert(rec.Body.String(), Equals, "404 Not found\n")
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
	c.Assert(rec.Code, Equals, 500)
	c.Assert(rec.Body.String(), Equals, "<!doctype html><html>50x</html>\n")
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
	c.Assert(rec.Body.String(), Equals, string(buf))
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
	c.Assert(rec.Code, Equals, 304)

	// Non-existing file
	req = customRequest(c, notFound)
	rec = httptest.NewRecorder()
	params[0].Value = notFoundTruncated
	serveImages(rec, req, params)
	c.Assert(rec.Code, Equals, 404)
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
	c.Assert(rec.Body.Bytes(), DeepEquals, desktop.HTML)
	c.Assert(rec.Header().Get("ETag"), Equals, desktop.Hash)

	// Mobile
	req = newRequest(c)
	req.Header.Set("User-Agent", mobileUA)
	rec = httptest.NewRecorder()
	serveIndexTemplate(rec, req)
	c.Assert(rec.Body.Bytes(), DeepEquals, mobile.HTML)
	c.Assert(rec.Header().Get("ETag"), Equals, mobile.Hash+"-mobile")

	// Etag matches
	req = newRequest(c)
	req.Header.Set("If-None-Match", desktop.Hash)
	rec = httptest.NewRecorder()
	serveIndexTemplate(rec, req)
	c.Assert(rec.Code, Equals, 304)
}

func (*DB) TestThreadHTML(c *C) {
	// Non-existant thread
	rec := httptest.NewRecorder()
	req := customRequest(c, "/a/22")
	body := []byte("body")
	resources = templateMap{
		"index": templateStore{
			HTML: body,
			Hash: "hash",
		},
	}
	webRoot = "./test"
	threadHTML("a")(rec, req, httprouter.Params{{Value: "22"}})
	c.Assert(rec.Code, Equals, 404)

	// Thread exists
	setupBoardAccess()
	setupPosts()
	rec = httptest.NewRecorder()
	req = customRequest(c, "/a/1")
	threadHTML("a")(rec, req, httprouter.Params{{Value: "1"}})
	c.Assert(rec.Body.Bytes(), DeepEquals, body)
}
