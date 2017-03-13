package server

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestAssetServer(t *testing.T) {
	t.Parallel()

	rec, req := newPair("/assets/frontpage.html")
	router.ServeHTTP(rec, req)
	assertBody(t, rec, "<!doctype html><html></html>\n")

	// Requesting a directory
	rec, req = newPair("/assets/js")
	router.ServeHTTP(rec, req)
}

func TestServeWorker(t *testing.T) {
	t.Parallel()

	workerPath = getWorkerPath()

	rec, req := newPair("/worker.js")
	router.ServeHTTP(rec, req)
	assertBody(t, rec, "console.log(\"Worker dess\")\n")
}

func TestImageServer(t *testing.T) {
	t.Parallel()

	const (
		found    = "/images/src/tis_life.gif"
		notFound = "/images/src/nobody_here.gif"
	)

	// Successful first serve
	rec, req := newPair(found)
	router.ServeHTTP(rec, req)
	path := filepath.Join("testdata", "src", "tis_life.gif")
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 {
		t.Fatal("failed to fetch image")
	}
	assertBody(t, rec, string(buf))
	assertHeaders(t, rec, imageHeaders)

	// Non-existing file
	rec, req = newPair(notFound)
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 404)
}
