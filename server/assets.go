package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

var (
	// Set of headers for serving images (and other uploaded files)
	imageHeaders = map[string]string{
		// max-age set to 350 days. Some caches and browsers ignore max-age, if it
		// is a year or greater, so keep it a little below.
		"Cache-Control": "max-age=30240000",

		// Fake etag to stop agressive browser cache busting
		"ETag":            "0",
		"X-Frame-Options": "sameorigin",
	}

	// For overriding during tests
	imageWebRoot = "images/"

	assetServer http.Handler
)

// More performant handler for serving image assets. These are immutable
// (except deletion), so we can also set seperate caching policies for them.
func serveImages(
	res http.ResponseWriter,
	req *http.Request,
	params map[string]string,
) {
	file, err := os.Open(filepath.FromSlash(imageWebRoot + params["path"]))
	if err != nil {
		text404(res, req)
		return
	}
	defer file.Close()

	if checkClientEtag(res, req, "0") {
		return
	}
	head := res.Header()
	for key, val := range imageHeaders {
		head.Set(key, val)
	}

	_, err = io.Copy(res, file)
	if err != nil {
		textErrorPage(res, req, err)
		return
	}
}

// Server static assets
func serveAssets(
	res http.ResponseWriter,
	req *http.Request,
	params map[string]string,
) {
	req.URL.Path = params["path"]
	assetServer.ServeHTTP(res, req)
}

// Serve the service worker script file. It needs to be on the root scope for
// security reasons.
func serveWorker(res http.ResponseWriter, req *http.Request) {
	path := filepath.FromSlash(webRoot + "/js/scripts/worker.js")
	http.ServeFile(res, req, path)
}
