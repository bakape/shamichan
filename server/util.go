package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/templates"
	"github.com/dimfeld/httptreemux"
	"github.com/go-playground/log"
)

const (
	// Body size limit for POST request JSON. Should never exceed 32 KB.
	// Consider anything bigger an attack.
	jsonLimit = 1 << 15
)

var (
	// Base set of HTTP headers for both HTML and JSON
	vanillaHeaders = map[string]string{
		"X-Frame-Options": "sameorigin",
		"Cache-Control":   "no-cache",
		"Expires":         "Fri, 01 Jan 1990 00:00:00 GMT",
	}
)

type uint64Sorter []uint64

func (p uint64Sorter) Len() int           { return len(p) }
func (p uint64Sorter) Less(i, j int) bool { return p[i] < p[j] }
func (p uint64Sorter) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

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
	log.Errorf(`server: ip="%s" url="%s" err="%s"`, ip, r.URL.String(), err)
}

// Text-only 404 response
func text404(w http.ResponseWriter) {
	http.Error(w, "404 not found", 404)
}

// Send error with code and logging according to error type
func httpError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

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
func assertNotBanned(w http.ResponseWriter, r *http.Request, board string,
) bool {
	ip, err := auth.GetIP(r)
	if err != nil {
		httpError(w, r, common.StatusError{
			Err:  err,
			Code: 400,
		})
		return false
	}
	err = db.IsBanned(board, ip)
	switch err {
	case nil:
		return true
	case common.ErrBanned:
	default:
		httpError(w, r, err)
		return false
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
		templates.WriteBanPage(w, rec)
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

// Decode JSON sent in a request with a read limit of 8 KB. Returns if the
// decoding succeeded.
func decodeJSON(r *http.Request, dest interface{}) (err error) {
	err = json.NewDecoder(io.LimitReader(r.Body, jsonLimit)).Decode(dest)
	if err != nil {
		err = common.StatusError{
			Err:  err,
			Code: 400,
		}
	}
	return
}

// Decode JSON post ID array from request body.
// Dedup and sort for faster DB access.
func decodePostIDArray(r *http.Request) (ids []uint64, err error) {
	err = decodeJSON(r, &ids)
	if err != nil {
		return
	}
	if len(ids) > 2 {
		m := make(map[uint64]struct{}, len(ids))
		for _, id := range ids {
			m[id] = struct{}{}
		}
		ids = ids[:0]
		for id := range m {
			ids = append(ids, id)
		}
		sort.Sort(uint64Sorter(ids))
	}
	return
}
