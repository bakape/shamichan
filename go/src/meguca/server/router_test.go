package server

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"

	"meguca/db"
	"meguca/lang"
	"meguca/templates"
	"meguca/util"

	"github.com/dimfeld/httptreemux"
)

// Global router used for tests
var router http.Handler

func init() {
	isTest = true
	router = createRouter()
	webRoot = "testdata"
	imageWebRoot = "testdata"
	db.ConnArgs = db.TestConnArgs
	db.IsTest = true

	err := util.Waterfall(db.LoadDB, lang.Load, templates.Compile)
	if err != nil {
		panic(err)
	}
}

func TestAllBoardRedirect(t *testing.T) {
	t.Parallel()

	rec, req := newPair("/")
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 302)

	loc := rec.Header().Get("Location")
	if loc != "/all/" {
		t.Fatalf("unexpected redirect result: %s", loc)
	}
}

func TestPanicHandler(t *testing.T) {
	t.Parallel()

	r := httptreemux.New()
	h := wrapHandler(func(_ http.ResponseWriter, _ *http.Request) {
		panic(errors.New("foo"))
	})
	r.GET("/panic", h)
	r.PanicHandler = text500
	rec, req := newPair("/panic")

	// Prevent printing stack trace to terminal
	log.SetOutput(ioutil.Discard)
	r.ServeHTTP(rec, req)
	log.SetOutput(os.Stdout)

	assertCode(t, rec, 500)
	assertBody(t, rec, "500 foo\n")
}

func TestGzip(t *testing.T) {
	enableGzip = true
	defer func() {
		enableGzip = false
	}()

	r := createRouter()
	rec, req := newPair("/json/config")
	req.Header.Set("Accept-Encoding", "gzip")

	r.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("response not gzipped")
	}
}
