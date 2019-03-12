package server

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/lang"
	"github.com/bakape/meguca/templates"
	"github.com/bakape/meguca/util"

	"github.com/dimfeld/httptreemux"
	"github.com/go-playground/log"
	"github.com/go-playground/log/handlers/console"
)

// Global router used for tests
var router http.Handler
var con = console.New(true)

func TestMain(m *testing.M) {
	close, err := db.LoadTestDB("server")
	if err != nil {
		panic(err)
	}

	log.AddHandler(con, log.AllLevels...)
	isTest = true
	router = createRouter()
	webRoot = "testdata"
	imageWebRoot = "testdata"

	config.Set(config.Configs{
		Public: config.Public{
			DefaultLang: "en_GB",
		},
	})
	if err := util.Waterfall(lang.Load, templates.Compile); err != nil {
		panic(err)
	}

	code := m.Run()
	err = close()
	if err != nil {
		panic(err)
	}
	os.Exit(code)
}

func TestAllBoardRedirect(t *testing.T) {
	t.Parallel()

	rec, req := newPair("/")
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 301)

	loc := rec.Header().Get("Location")
	if loc != "/all/" {
		t.Fatalf("unexpected redirect result: %s", loc)
	}
}

func TestPanicHandler(t *testing.T) {
	r := httptreemux.NewContextMux()
	h := func(_ http.ResponseWriter, _ *http.Request) {
		panic(errors.New("foo"))
	}
	r.GET("/panic", h)
	r.PanicHandler = handlePanic
	rec, req := newPair("/panic")

	// Prevent printing stack trace to terminal
	con.SetWriter(ioutil.Discard)
	defer con.SetWriter(os.Stdout)

	r.ServeHTTP(rec, req)
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
