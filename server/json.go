package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/cache"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
	"github.com/bakape/meguca/websockets/feeds"
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
		httpError(w, r, err)
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

	setJSONHeaders(w)
	w.Header().Set("ETag", etag)

	writeData(w, r, buf)
}

func setJSONHeaders(w http.ResponseWriter) {
	head := w.Header()
	for key, val := range vanillaHeaders {
		head.Set(key, val)
	}
	head.Set("Content-Type", "application/json")
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
		httpError(w, r, common.StatusError{
			Err:  err,
			Code: 400,
		})
		return
	}

	post, err := db.GetPost(id)
	if err != nil {
		httpError(w, r, err)
		return
	}
	serveJSON(w, r, "", post)
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
	var (
		page   int
		thread uint64
	)
	ok := func() (ok bool) {
		var err error
		thread, err = strconv.ParseUint(extractParam(r, "thread"), 10, 64)
		if err != nil {
			return
		}

		if s := r.URL.Query().Get("page"); s != "" {
			page, err = strconv.Atoi(s)
			if err != nil || page < -1 {
				return
			}
		}

		ok = true
		return
	}()
	if !ok {
		text404(w)
		return
	}

	httpError(w, r, func() (err error) {
		setJSONHeaders(w)
		return cache.Thread(w, r, thread, page)
	}())
}

// Serves board page JSON
func boardJSON(w http.ResponseWriter, r *http.Request, catalog bool) {
	var (
		page  uint64
		board string
	)
	ok := func() (ok bool) {
		board = extractParam(r, "board")
		if !auth.IsBoard(board) {
			return
		}

		if s := r.URL.Query().Get("page"); s != "" {
			var err error
			page, err = strconv.ParseUint(s, 10, 32)
			if err != nil {
				return
			}
		}

		ok = true
		return
	}()
	if !ok {
		text404(w)
		return
	}

	httpError(w, r, func() (err error) {
		setJSONHeaders(w)
		return cache.Board(w, r, board, uint32(page))
	}())
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

func serveThreadUpdates(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		var data map[uint64]uint64
		err = decodeJSON(r, &data)
		if err != nil {
			return
		}

		diff, err := db.DiffThreadPostCounts(data)
		if err != nil {
			return
		}
		serveJSON(w, r, "", diff)
		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}
