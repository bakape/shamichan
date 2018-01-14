package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"meguca/auth"
	"meguca/cache"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"meguca/util"
	"meguca/websockets/feeds"
	"net/http"
	"strconv"
)

var errNoImage = errors.New("post has no image")

// Request to spoiler an already allocated image that the sender has created
type spoilerRequest struct {
	ID       uint64
	Password string
}

// Marshal input data to JSON an write to client
func serveJSON(
	w http.ResponseWriter,
	r *http.Request,
	etag string,
	data interface{},
) {
	buf, err := json.Marshal(data)
	if err != nil {
		text500(w, r, err)
		return
	}
	writeJSON(w, r, etag, buf)
}

// Write data as JSON to the client. If etag is "" generate a strong etag by
// hashing the resulting buffer and perform a check against the "If-None-Match"
// header. If etag is set, assume this check has already been done.
func writeJSON(
	w http.ResponseWriter,
	r *http.Request,
	etag string,
	buf []byte,
) {
	if etag == "" {
		etag = util.HashBuffer(buf)
	}
	if checkClientEtag(w, r, etag) {
		return
	}

	head := w.Header()
	for key, val := range vanillaHeaders {
		head.Set(key, val)
	}
	head.Set("ETag", etag)
	head.Set("Content-Type", "application/json")

	writeData(w, r, buf)
}

// Validate the client's last N posts to display setting. To allow for better
// caching the only valid values are 5 and 50. 5 is for index-like thread
// previews and 50 is for short threads.
func detectLastN(r *http.Request) int {
	if q := r.URL.Query().Get("last"); q != "" {
		n, err := strconv.Atoi(q)
		if err == nil && (n == 100 || n == 5) {
			return n
		}
	}
	return 0
}

// Serve public configuration information as JSON
func serveConfigs(w http.ResponseWriter, r *http.Request) {
	buf, etag := config.GetClient()
	writeJSON(w, r, etag, buf)
}

// Serve a single post as JSON
func servePost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(extractParam(r, "post"), 10, 64)
	if err != nil {
		text400(w, err)
		return
	}

	switch post, err := db.GetPost(id); err {
	case nil:
		serveJSON(w, r, "", post)
	case sql.ErrNoRows:
		text404(w)
	default:
		respondToJSONError(w, r, err)
	}
}

func respondToJSONError(w http.ResponseWriter, r *http.Request, err error) {
	if err == sql.ErrNoRows {
		text404(w)
	} else {
		text500(w, r, err)
	}
}

// Serve board-specific configuration JSON
func serveBoardConfigs(
	w http.ResponseWriter,
	r *http.Request,
) {
	board := extractParam(r, "board")
	if !auth.IsBoard(board) {
		text404(w)
		return
	}

	conf := config.GetBoardConfigs(board)
	if conf.ID == "" { // Data race with DB. Board deleted.
		text404(w)
		return
	}
	writeJSON(w, r, conf.Hash, conf.JSON)
}

// Serves thread page JSON
func threadJSON(w http.ResponseWriter, r *http.Request) {
	id, ok := validateThread(w, r)
	if !ok {
		return
	}

	k := cache.ThreadKey(id, detectLastN(r))
	data, _, ctr, err := cache.GetJSONAndData(k, cache.ThreadFE)
	if err != nil {
		respondToJSONError(w, r, err)
		return
	}

	writeJSON(w, r, formatEtag(ctr, "", auth.NotLoggedIn), data)
}

// Confirms a the thread exists on the board and returns its ID. If an error
// occurred and the calling function should return, ok = false.
func validateThread(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	board := extractParam(r, "board")

	if !assertNotBanned(w, r, board) {
		return 0, false
	}

	id, err := strconv.ParseUint(extractParam(r, "thread"), 10, 64)
	if err != nil {
		text404(w)
		return 0, false
	}

	valid, err := db.ValidateOP(id, board)
	if err != nil {
		text500(w, r, err)
		return 0, false
	}
	if !valid {
		text404(w)
		return 0, false
	}

	return id, true
}

// Serves board page JSON
func boardJSON(w http.ResponseWriter, r *http.Request, catalog bool) {
	b := extractParam(r, "board")
	if !auth.IsBoard(b) {
		text404(w)
		return
	}
	if !assertNotBanned(w, r, b) {
		return
	}

	data, _, ctr, err := cache.GetJSONAndData(boardCacheArgs(r, b, catalog))
	switch err {
	case nil:
		writeJSON(w, r, formatEtag(ctr, "", auth.NotLoggedIn), data)
	case cache.ErrPageOverflow:
		text404(w)
	default:
		text500(w, r, err)
	}
}

// Serve a JSON array of all available boards and their titles
func serveBoardList(res http.ResponseWriter, req *http.Request) {
	serveJSON(res, req, "", config.GetBoardTitles())
}

// Serve map of internal file type enums to extensions. Needed for
// version-independent backwards compatibility with external applications.
func serveExtensionMap(w http.ResponseWriter, r *http.Request) {
	serveJSON(w, r, "", common.Extensions)
}

// Serve number of unique connected IPs
func serveIPCount(w http.ResponseWriter, r *http.Request) {
	serveJSON(w, r, "", feeds.IPCount())
}
