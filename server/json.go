package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/server/websockets"
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
// hashing the resulting buffer and perorm a check against the "If-None-Match"
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
// previews and 50 is for short theads.
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
	if board == "all" {
		if !checkClientEtag(w, r, "0") {
			writeJSON(w, r, "0", config.AllBoardConfigs)
		}
		return
	}
	if !auth.IsNonMetaBoard(board) {
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
	board := p["board"]
	id, err := strconv.ParseInt(p["thread"], 10, 64)
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

	counter, err := db.ThreadCounter(id)
	if err != nil {
		text500(w, r, err)
		return
	}
	etag := etagStart(counter)
	if checkClientEtag(w, r, etag) {
		return
	}

	data, err := db.GetThread(id, detectLastN(r))
	if err != nil {
		respondToJSONError(w, r, err)
		return
	}
	serveJSON(w, r, etag, data)
}

// Serves JSON for the "/all/" meta-board, that contains threads from all boards
func allBoardJSON(w http.ResponseWriter, r *http.Request) {
	counter, err := db.PostCounter()
	if err != nil {
		text500(w, r, err)
		return
	}
	etag := etagStart(counter)
	if checkClientEtag(w, r, etag) {
		return
	}

	data, err := db.GetAllBoard()
	if err != nil {
		text500(w, r, err)
		return
	}
	serveJSON(w, r, etag, data)
}

// Serves board page JSON
func boardJSON(w http.ResponseWriter, r *http.Request, p map[string]string) {
	board := p["board"]
	if !auth.IsBoard(board) {
		text404(w)
		return
	}

	counter, err := db.BoardCounter(board)
	if err != nil {
		text500(w, r, err)
		return
	}
	etag := etagStart(counter)
	if checkClientEtag(w, r, etag) {
		return
	}

	data, err := db.GetBoard(board)
	if err != nil {
		text500(w, r, err)
		return
	}
	serveJSON(w, r, etag, data)
}

// Serve a JSON array of all available boards and their titles
func serveBoardList(res http.ResponseWriter, req *http.Request) {
	type boardEntries []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}

	var list boardEntries
	q := r.Table("boards").Pluck("id", "title")
	if err := db.All(q, &list); err != nil {
		text500(res, req, err)
		return
	}
	if list == nil { // Ensure always serving an array
		list = boardEntries{}
	}
	serveJSON(res, req, "", list)
}

// Fetch an array of boards a certain user holds a certion position on
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

// Spoiler an already allocated image
func spoilerImage(w http.ResponseWriter, req *http.Request) {
	var msg spoilerRequest
	if !decodeJSON(w, req, &msg) {
		return
	}

	var res struct {
		Image    types.Image
		Password []byte
	}
	q := db.FindPost(msg.ID).Pluck("image", "password").Default(nil)
	if err := db.One(q, &res); err != nil {
		text500(w, req, err)
		return
	}

	if res.Image.SHA1 == "" {
		text400(w, errNoImage)
		return
	}
	if res.Image.Spoiler { // Already spoilered. NOOP.
		return
	}
	if err := auth.BcryptCompare(msg.Password, res.Password); err != nil {
		text403(w, err)
		return
	}

	logMsg, err := websockets.EncodeMessage(websockets.MessageSpoiler, msg.ID)
	if err != nil {
		text500(w, req, err)
		return
	}

	update := map[string]bool{
		"spoiler": true,
	}
	err = websockets.UpdatePost(msg.ID, "image", update, logMsg)
	if err != nil {
		text500(w, req, err)
		return
	}
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
