package server

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"strconv"
)

const assetCacheHeader = "max-age=0, must-revalidate"

var (
	// Set of headers for serving images (and other uploaded files)
	imageHeaders = map[string]string{
		// max-age set to 350 days. Some caches and browsers ignore max-age, if
		// it is a year or greater, so keep it a little below.
		"Cache-Control":   "max-age=30240000, public, immutable",
		"X-Frame-Options": "sameorigin",
	}

	// For overriding during tests
	imageWebRoot = "images"

	// Path to the service worker script. Overrideable in tests.
	workerPath = getWorkerPath()
)

func getWorkerPath() string {
	return filepath.FromSlash(webRoot + "/js/scripts/worker.js")
}

// More performant handler for serving image assets. These are immutable
// (except deletion), so we can also set separate caching policies for them.
func serveImages(w http.ResponseWriter, r *http.Request, p map[string]string) {
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

	stats, err := file.Stat()
	if err != nil {
		text500(w, r, err)
		return
	}
	if stats.IsDir() {
		text404(w)
		return
	}
	modTime := stats.ModTime()
	etag := strconv.FormatInt(modTime.Unix(), 10)

	head := w.Header()
	head.Set("Cache-Control", assetCacheHeader)
	head.Set("ETag", etag)
	http.ServeContent(w, r, path, modTime, file)
}

// Serve the service worker script file. It needs to be on the root scope for
// security reasons.
func serveWorker(w http.ResponseWriter, r *http.Request) {
	serveFile(w, r, workerPath)
}
