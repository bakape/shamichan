// Webserver

package server

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/util"
)

// Base set of HTTP headers for both HTML and JSON
var vanillaHeaders = map[string]string{
	"X-Frame-Options": "sameorigin",
	"Cache-Control":   "max-age=0, must-revalidate",
	"Expires":         "Fri, 01 Jan 1990 00:00:00 GMT",
}

// Build an etag for HTML or JSON pages and check if it matches the one provided
// by the client. If yes, send 304 and return false, otherwise set headers and
// return true.
func pageEtag(w http.ResponseWriter, r *http.Request, etag string) bool {
	// If etags match, no need to rerender
	if checkClientEtag(w, r, etag) {
		return false
	}
	setHeaders(w, etag)
	return true
}

// Build the main part of the etag
func etagStart(counter int64) string {
	return "W/" + util.IDToString(counter)
}

// Check is any of the etags the client provides in the "If-None-Match" header
// match the generated etag. If yes, write 304 and return true.
func checkClientEtag(
	w http.ResponseWriter,
	r *http.Request,
	etag string,
) bool {
	if etag == r.Header.Get("If-None-Match") {
		w.WriteHeader(304)
		return true
	}
	return false
}

// Write a []byte to the client
func writeData(w http.ResponseWriter, r *http.Request, data []byte) {
	_, err := w.Write(data)
	if err != nil {
		logError(r, err)
	}
}

// Log an error together with the client's IP and stack trace
func logError(r *http.Request, err interface{}) {
	if !isTest { // Do not polute test output with logs
		log.Printf("server: %s: %s\n%s", auth.GetIP(r), err, debug.Stack())
	}
}

// Set HTTP headers to the response object
func setHeaders(w http.ResponseWriter, etag string) {
	head := w.Header()
	for key, val := range vanillaHeaders {
		head.Set(key, val)
	}
	head.Set("ETag", etag)
}

// Text-only 404 response
func text404(w http.ResponseWriter) {
	http.Error(w, "404 Not found", 404)
}

// Text-only 400 response
func text400(w http.ResponseWriter, err error) {
	http.Error(w, fmt.Sprintf("400 Bad request: %s", err), 400)
}

func text403(w http.ResponseWriter, err error) {
	http.Error(w, fmt.Sprintf("403 Forbidden: %s", err), 403)
}

// Text-only 500 response
func text500(w http.ResponseWriter, r *http.Request, err interface{}) {
	http.Error(w, fmt.Sprintf("500 Internal server error: %s", err), 500)
	logError(r, err)
}
