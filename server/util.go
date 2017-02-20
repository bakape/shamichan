package server

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"../auth"
	"../db"
	"../lang"
	"../templates"
)

// Base set of HTTP headers for both HTML and JSON
var vanillaHeaders = map[string]string{
	"X-Frame-Options": "sameorigin",
	"Cache-Control":   "max-age=0, must-revalidate",
	"Expires":         "Fri, 01 Jan 1990 00:00:00 GMT",
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
	log.Printf("server: %s: %s\n%s", auth.GetIP(r), err, debug.Stack())
}

// Text-only 404 response
func text404(w http.ResponseWriter) {
	http.Error(w, "404 not found", 404)
}

// Text-only 400 response
func text400(w http.ResponseWriter, err error) {
	http.Error(w, fmt.Sprintf("400 %s", err), 400)
}

func text403(w http.ResponseWriter, err error) {
	http.Error(w, fmt.Sprintf("403 %s", err), 403)
}

// Text-only 500 response
func text500(w http.ResponseWriter, r *http.Request, err interface{}) {
	http.Error(w, fmt.Sprintf("500 %s", err), 500)
	logError(r, err)
}

// Check client is not banned on specific board. Returns true, if all clear.
// Renders ban page and returns false otherwise.
func assertNotBanned(
	w http.ResponseWriter,
	r *http.Request,
	board string,
) bool {
	ip := auth.GetIP(r)
	if !auth.IsBanned(board, ip) {
		return true
	}

	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return false
	}

	rec, err := db.GetBanInfo(ip, board)
	if err != nil {
		text500(w, r, err)
		return false
	}
	w.WriteHeader(403)
	head := w.Header()
	for key, val := range vanillaHeaders {
		head.Set(key, val)
	}
	head.Set("Content-Type", "text/html")
	html := []byte(templates.BanPage(rec, lp.Templates["banPage"]))
	w.Write(html)
	return false
}
