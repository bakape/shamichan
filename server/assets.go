package server

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bakape/meguca/util"
)

const assetCacheHeader = "max-age=0, must-revalidate"

var (
	// Set of headers for serving images (and other uploaded files)
	imageHeaders = map[string]string{
		// max-age set to 350 days. Some caches and browsers ignore max-age, if it
		// is a year or greater, so keep it a little below.
		"Cache-Control":   "max-age=30240000, public",
		"X-Frame-Options": "sameorigin",

		// Fake E-tag, because all images are immutable
		"ETag": "0",
	}

	// For overriding during tests
	imageWebRoot = "images"
)

// More performant handler for serving image assets. These are immutable
// (except deletion), so we can also set seperate caching policies for them.
func serveImages(w http.ResponseWriter, r *http.Request, p map[string]string) {
	if r.Header.Get("If-None-Match") == "0" {
		w.WriteHeader(304)
		return
	}

	file, err := os.Open(cleanJoin(imageWebRoot, p["path"]))
	if err != nil {
		text404(w)
		return
	}
	defer file.Close()

	head := w.Header()
	for key, val := range imageHeaders {
		head.Set(key, val)
	}

	http.ServeContent(w, r, p["path"], time.Time{}, file)
}

func cleanJoin(a, b string) string {
	return filepath.Clean(filepath.Join(a, b))
}

// Server static assets
func serveAssets(w http.ResponseWriter, r *http.Request, p map[string]string) {
	serveFile(w, r, cleanJoin(webRoot, p["path"]))
}

func serveFile(w http.ResponseWriter, r *http.Request, path string) {
	file, err := os.Open(path)
	if err != nil {
		text404(w)
		return
	}
	defer file.Close()

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		text500(w, r, err)
		return
	}

	head := w.Header()
	head.Set("Cache-Control", assetCacheHeader)
	head.Set("ETag", util.HashBuffer(buf))

	http.ServeContent(w, r, path, time.Time{}, bytes.NewReader(buf))
}

// Serve the service worker script file. It needs to be on the root scope for
// security reasons.
func serveWorker(w http.ResponseWriter, r *http.Request) {
	serveFile(w, r, filepath.FromSlash(webRoot+"/js/scripts/worker.js"))
}
