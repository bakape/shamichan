package server

import (
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	// Set of headers for serving images (and other uploaded files)
	imageHeaders = map[string]string{
		// max-age set to 350 days. Some caches and browsers ignore max-age, if it
		// is a year or greater, so keep it a little below.
		"Cache-Control":   "max-age=30240000",
		"X-Frame-Options": "sameorigin",
	}

	// For overriding during tests
	imageWebRoot = "images/"

	assetServer http.Handler
)

// More performant handler for serving image assets. These are immutable
// (except deletion), so we can also set seperate caching policies for them.
func serveImages(w http.ResponseWriter, r *http.Request, p map[string]string) {
	file, err := os.Open(filepath.FromSlash(imageWebRoot + p["path"]))
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

// Server static assets
func serveAssets(w http.ResponseWriter, r *http.Request, p map[string]string) {
	r.URL.Path = p["path"]
	w.Header().Set("Cache-Control", "max-age=0,must-revalidate")
	assetServer.ServeHTTP(w, r)
}

// Serve the service worker script file. It needs to be on the root scope for
// security reasons.
func serveWorker(res http.ResponseWriter, req *http.Request) {
	path := filepath.FromSlash(webRoot + "/js/scripts/worker.js")
	http.ServeFile(res, req, path)
}
