package server

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/bakape/meguca/cache"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/lang"
	"github.com/dimfeld/httptreemux"
	"github.com/go-playground/log"
	"github.com/go-playground/log/handlers/console"
)

// Global router used for tests
var router http.Handler
var con = console.New(true)

func TestMain(m *testing.M) {
	code := 1
	err := func() (err error) {
		err = config.Server.Load()
		if err != nil {
			return
		}
		err = db.LoadTestDB()
		if err != nil {
			return
		}

		log.AddHandler(con, log.AllLevels...)
		router = createRouter()
		webRoot = "testdata"
		imageWebRoot = "testdata"

		config.Set(config.Configs{
			Public: config.Public{
				DefaultLang: "en_GB",
			},
		})
		config.Server.CacheSize = 100
		err = cache.Init()
		if err != nil {
			return
		}
		err = lang.Load()
		if err != nil {
			return
		}

		code = m.Run()
		return
	}()
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
