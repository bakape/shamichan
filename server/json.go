package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

var (
	errNoImage = errors.New("post has no image")
)

// Request to spoiler an already allocated image that the sender has created
type spoilerRequest struct {
	ID       int64
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
		if checkClientEtag(w, r, etag) {
			return
		}
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
func detectLastN(req *http.Request) int {
	if q := req.URL.Query().Get("last"); q != "" {
		n, err := strconv.Atoi(q)
		if err == nil && (n == 50 || n == 5) {
			return n
		}
	}
	return 0
}

// Serve public configuration information as JSON
func serveConfigs(w http.ResponseWriter, r *http.Request) {
	buf, etag := config.GetClient()
	if checkClientEtag(w, r, etag) {
		return
	}
	writeJSON(w, r, etag, buf)
}

// Serve a single post as JSON
func servePost(w http.ResponseWriter, r *http.Request, p map[string]string) {
	id, err := strconv.ParseInt(p["post"], 10, 64)
	if err != nil {
		text400(w, err)
		return
	}

	post, err := db.GetPost(id)
	if err != nil {
		respondToJSONError(w, r, err)
		return
	}

	serveJSON(w, r, "", post)
}

func respondToJSONError(w http.ResponseWriter, req *http.Request, err error) {
	if err == r.ErrEmptyResult {
		text404(w)
	} else {
		text500(w, req, err)
	}
}

// Serve board-specific configuration JSON
func serveBoardConfigs(
	w http.ResponseWriter,
	r *http.Request,
	p map[string]string,
) {
	board := p["board"]
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
func threadJSON(w http.ResponseWriter, r *http.Request, p map[string]string) {
	id, ok := validateThread(w, r, p)
	if !ok {
		return
	}
	data, etag, ok := threadData(w, r, id)
	if !ok {
		return
	}

	serveJSON(w, r, etag, data)
}

// Confirms a the thread exists on the board and returns its ID. If an error
// occurred and the calling function should return, ok = false.
func validateThread(
	w http.ResponseWriter,
	r *http.Request,
	p map[string]string,
) (id int64, ok bool) {
	board := p["board"]
	var err error
	id, err = strconv.ParseInt(p["thread"], 10, 64)
	if err != nil {
		text400(w, err)
		return
	}

	valid, err := db.ValidateOP(id, board)
	if err != nil {
		text500(w, r, err)
		return
	}
	if !valid {
		text404(w)
		return
	}

	return id, true
}

// Retrieves thread data from the database and the associated etag header. If an
// error occurred and the calling function should return, ok = false.
func threadData(w http.ResponseWriter, r *http.Request, id int64) (
	data *types.Thread, etag string, ok bool,
) {
	counter, err := db.ThreadCounter(id)
	if err != nil {
		text500(w, r, err)
		return
	}
	etag = etagStart(counter)
	if checkClientEtag(w, r, etag) {
		return
	}

	data, err = db.GetThread(id, detectLastN(r))
	if err != nil {
		respondToJSONError(w, r, err)
		return
	}

	return data, etag, true
}

// Serves board page JSON
func boardJSON(w http.ResponseWriter, r *http.Request, p map[string]string) {
	b := p["board"]
	if !auth.IsBoard(b) {
		text404(w)
		return
	}

	data, etag, ok := boardData(w, r, b)
	if !ok {
		return
	}
	serveJSON(w, r, etag, data)
}

// Retrieves board data from the database and the associated etag header. If an
// error occurred and the calling function should return, ok = false.
func boardData(w http.ResponseWriter, r *http.Request, b string) (
	data *types.Board, etag string, ok bool,
) {
	if b == "all" {
		return allBoardData(w, r)
	}

	counter, err := db.BoardCounter(b)
	if err != nil {
		text500(w, r, err)
		return
	}
	etag = etagStart(counter)
	if checkClientEtag(w, r, etag) {
		return
	}

	data, err = db.GetBoard(b)
	if err != nil {
		text500(w, r, err)
		return
	}

	return data, etag, true
}

// Same as boardData(), but for the /all/ metaboard
func allBoardData(w http.ResponseWriter, r *http.Request) (
	data *types.Board, etag string, ok bool,
) {
	counter, err := db.PostCounter()
	if err != nil {
		text500(w, r, err)
		return
	}
	etag = etagStart(counter)
	if checkClientEtag(w, r, etag) {
		return
	}

	data, err = db.GetAllBoard()
	if err != nil {
		text500(w, r, err)
		return
	}

	return data, etag, true
}

// Serve a JSON array of all available boards and their titles
func serveBoardList(res http.ResponseWriter, req *http.Request) {
	serveJSON(res, req, "", config.GetBoardTitles())
}

// Fetch an array of boards a certain user holds a certain position on
func serveStaffPositions(
	res http.ResponseWriter,
	req *http.Request,
	params map[string]string,
) {
	q := r.
		Table("boards").
		Filter(r.Row.
			Field("staff").
			Field(params["position"]).
			Contains(params["user"]),
		).
		Field("id").
		CoerceTo("array")
	var boards []string
	if err := db.All(q, &boards); err != nil {
		text500(res, req, err)
		return
	}

	// Ensure response is always a JSON array
	if boards == nil {
		boards = []string{}
	}

	serveJSON(res, req, "", boards)
}

// Serve boards' last update timestamps. A board with no  posts will produce
// zero.
func serveBoardTimestamps(w http.ResponseWriter, req *http.Request) {
	q := r.
		Expr(config.GetBoards()).
		Map(func(b r.Term) r.Term {
			return r.Object(
				b,
				r.
					Table("posts").
					GetAllByIndex("board", b).
					Field("lastUpdated").
					Max().
					Default(0),
			)
		}).
		Reduce(func(a, b r.Term) r.Term {
			return a.Merge(b)
		})
	var ctrs map[string]uint64
	if err := db.One(q, &ctrs); err != nil {
		text500(w, req, err)
		return
	}
	serveJSON(w, req, "", ctrs)
}
