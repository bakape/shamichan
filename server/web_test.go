package server

import (
	. "gopkg.in/check.v1"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type WebServer struct{}

var _ = Suite(&WebServer{})

func (w *WebServer) TestFrontpageRedirect(c *C) {
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

func (w *WebServer) TestDefaultBoardRedirect(c *C) {
	config = serverConfigs{}
	config.Boards.Default = "a"
	rec := w.runHandler(c, redirectToDefault)
	c.Assert(rec.Code, Equals, 302)
	c.Assert(rec.Header().Get("Location"), Equals, "/a/")
}

func (w *WebServer) runHandler(
	c *C,
	h http.HandlerFunc,
) *httptest.ResponseRecorder {
	req, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, IsNil)
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func (w *WebServer) TestConfigServing(c *C) {
	configHash = "foo"
	clientConfig = clientConfigs{}
	etag := "W/" + configHash
	rec := w.runHandler(c, serveConfigs)
	c.Assert(rec.Code, Equals, 200)
	c.Assert(rec.Body.String(), Equals, string(marshalJSON(clientConfig)))
	c.Assert(rec.Header().Get("ETag"), Equals, etag)

	// And with etag
	rec = httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, IsNil)
	req.Header.Set("If-None-Match", etag)
	serveConfigs(rec, req)
	c.Assert(rec.Code, Equals, 304)
}

func (w *WebServer) TestEtagComparison(c *C) {
	req, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, IsNil)
	const etag = "foo"
	req.Header.Set("If-None-Match", etag)
	rec := httptest.NewRecorder()
	c.Assert(checkClientEtags(rec, req, etag), Equals, true)
}

func (w *WebServer) TestNotFoundHandler(c *C) {
	rec := w.runHandler(c, notFound)
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
	for key, val := range headers {
		c.Assert(rec.Header().Get(key), Equals, val)
	}
}
