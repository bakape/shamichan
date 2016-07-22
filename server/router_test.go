package server

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/bakape/meguca/config"
	"github.com/dimfeld/httptreemux"
	. "gopkg.in/check.v1"
)

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

func (w *WebServer) TestAllBoardRedirect(c *C) {
	rec := httptest.NewRecorder()
	req := newRequest(c, "/")
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 302, c)
	c.Assert(rec.Header().Get("Location"), Equals, "/all/")
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

func (w *WebServer) TestGzip(c *C) {
	enableGzip = true
	r := createRouter()
	rec, req := newPair(c, "/json/config")
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(rec, req)
	c.Assert(rec.Header().Get("Content-Encoding"), Equals, "gzip")
}
