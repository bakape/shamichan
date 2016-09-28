package server

import (
	"io/ioutil"

	. "gopkg.in/check.v1"
)

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

func (*WebServer) TestImageServer(c *C) {
	const (
		truncated         = "/src/tis life.gif"
		notFoundTruncated = "src/nobody here.gif"
	)
	imageWebRoot = "testdata"
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
	assertHeaders(c, rec, imageHeaders)

	// Second fetch
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
