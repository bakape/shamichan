package server

import (
	"database/sql"
	"fmt"
	"meguca/auth"
	"meguca/common"
	"meguca/db"
	"meguca/templates"
	"net/http"
	"strconv"

	"github.com/dimfeld/httptreemux"
	"github.com/go-playground/log"
)

// Base set of HTTP headers for both HTML and JSON
var vanillaHeaders = map[string]string{
	"X-Frame-Options": "sameorigin",
	"Cache-Control":   "no-cache",
	"Expires":         "Fri, 01 Jan 1990 00:00:00 GMT",
}

// Check if the etag the client provides in the "If-None-Match" header matches
// the generated etag. If yes, write 304 and return true.
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

// Combine the progress counter and optional configuration hash into a weak etag
func formatEtag(ctr uint64, hash string, pos auth.ModerationLevel) string {
	buf := append(make([]byte, 0, 128), "W/\""...)
	buf = strconv.AppendUint(buf, ctr, 10)

	addOpt := func(s string) {
		buf = append(buf, '-')
		buf = append(buf, s...)
	}
	if hash != "" {
		addOpt(hash)
	}
	if pos != auth.NotLoggedIn {
		addOpt(pos.String())
	}

	return string(append(buf, '"'))
}

// Write a []byte to the client. Must receive the entire response body at once.
func writeData(w http.ResponseWriter, r *http.Request, data []byte) {
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, err := w.Write(data)
	if err != nil {
		logError(r, err)
	}
}

// Log an error together with the client's IP
func logError(r *http.Request, err interface{}) {
	if err, ok := err.(error); ok && common.CanIgnoreClientError(err) {
		return
	}

	ip, ipErr := auth.GetIP(r)
	if ipErr != nil {
		ip = "invalid IP"
	}
	log.Errorf("server: by %s: %s: %#v", ip, err, err)
}

// Text-only 404 response
func text404(w http.ResponseWriter) {
	http.Error(w, "404 not found", 404)
}

// Send error with code and logging according to error type
func httpError(w http.ResponseWriter, r *http.Request, err error) {
	code := 500
	switch err.(type) {
	case common.StatusError:
		code = err.(common.StatusError).Code
	default:
		if err == sql.ErrNoRows {
			code = 404
		}
	}

	http.Error(w, fmt.Sprintf("%d %s", code, err), code)
	if code >= 500 && code < 600 {
		logError(r, err)
	}
}

// Check client is not banned on specific board. Returns true, if all clear.
// Renders ban page and returns false otherwise.
func assertNotBanned(
	w http.ResponseWriter,
	r *http.Request,
	board string,
) bool {
	ip, err := auth.GetIP(r)
	if err != nil {
		httpError(w, r, common.StatusError{err, 400})
		return false
	}
	globally, fromBoard := auth.GetBannedLevels(board, ip)
	if !globally && !fromBoard {
		return true
	}
	if globally {
		board = "all"
	}

	rec, err := db.GetBanInfo(ip, board)
	switch err {
	case nil:
		w.WriteHeader(403)
		head := w.Header()
		for key, val := range vanillaHeaders {
			head.Set(key, val)
		}
		head.Set("Content-Type", "text/html")
		head.Set("Cache-Control", "no-store")
		html := []byte(templates.BanPage(rec))
		w.Write(html)
		return false
	case sql.ErrNoRows:
		// If there is no row, that means the ban cache has not been updated
		// yet with a cleared ban. Force a ban cache refresh.
		if err := db.RefreshBanCache(); err != nil {
			log.Errorf("refreshing ban cache: %s", err)
		}
		return true
	default:
		httpError(w, r, err)
		return false
	}
}

// Extract URL paramater from request context
func extractParam(r *http.Request, id string) string {
	return httptreemux.ContextParams(r.Context())[id]
}
