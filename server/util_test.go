package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "gopkg.in/check.v1"
)

const (
	notFound = "404 Not found"
)

func Test(t *testing.T) { TestingT(t) }

func dummyLog(n int) [][]byte {
	log := make([][]byte, n)
	for i := 0; i < n; i++ {
		log[i] = []byte{1}
	}
	return log
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

func (w *WebServer) TestText404(c *C) {
	rec, req := newPair(c, "/lalala/")
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)
	assertBody(rec, notFound, c)
}

func (w *WebServer) TestText500(c *C) {
	rec, req := newPair(c, "/")
	text404(rec, req)
	assertCode(rec, 404, c)
	assertBody(rec, "404 Not found", c)
}
