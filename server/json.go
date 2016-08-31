package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

// Serve JSON retrieved from the database and handle ETag-related functionality
func serveJSON(
	res http.ResponseWriter,
	req *http.Request,
	data []byte,
	err error,
) {
	if err != nil {
		text500(res, req, err)
		return
	}

	etag := util.HashBuffer(data)
	if checkClientEtag(res, req, etag) {
		return
	}
	setHeaders(res, etag)
	setJSONCType(res)
	writeData(res, req, data)
}

// Convert input data to JSON an write to client
func writeJSON(
	w http.ResponseWriter,
	r *http.Request,
	setEtag bool,
	data interface{},
) {
	buf, err := json.Marshal(data)
	if err != nil {
		text500(w, r, err)
		return
	}
	setJSONCType(w)
	if setEtag {
		w.Header().Set("ETag", util.HashBuffer(buf))
	}
	writeData(w, r, buf)
}

func setJSONCType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

// Validate the client's last N posts to display setting
func detectLastN(req *http.Request) int {
	query := req.URL.Query().Get("lastN")
	if query != "" {
		lastN, err := strconv.Atoi(query)
		if err == nil && lastN <= 500 {
			return lastN
		}
	}
	return 0
}

// Serve public configuration information as JSON
func serveConfigs(res http.ResponseWriter, req *http.Request) {
	json, etag := config.GetClient()
	if checkClientEtag(res, req, etag) {
		return
	}
	setHeaders(res, etag)
	setJSONCType(res)
	writeData(res, req, json)
}

// Serve a single post as JSON
func servePost(
	res http.ResponseWriter,
	req *http.Request,
	params map[string]string,
) {
	id, err := strconv.ParseInt(params["post"], 10, 64)
	if err != nil {
		text404(res)
		return
	}

	post, err := db.GetPost(id)
	if err != nil {
		respondToJSONError(res, req, err)
		return
	}

	data, err := json.Marshal(post)
	serveJSON(res, req, data, err)
}

func respondToJSONError(res http.ResponseWriter, req *http.Request, err error) {
	if err == r.ErrEmptyResult {
		text404(res)
	} else {
		text500(res, req, err)
	}
}

// Serve board-specific configuration JSON
func serveBoardConfigs(
	res http.ResponseWriter,
	req *http.Request,
	params map[string]string,
) {
	board := params["board"]
	if board == "all" {
		serveJSON(res, req, config.AllBoardConfigs, nil)
		return
	}
	if !auth.IsNonMetaBoard(board) {
		text404(res)
		return
	}

	var conf config.BoardConfigs
	if err := db.One(db.GetBoardConfig(board), &conf); err != nil {
		text500(res, req, err)
		return
	}

	data, err := conf.MarshalPublicJSON()
	serveJSON(res, req, data, err)
}

// Serves thread page JSON
func threadJSON(
	res http.ResponseWriter,
	req *http.Request,
	params map[string]string,
) {
	board := params["board"]
	id, err := strconv.ParseInt(params["thread"], 10, 64)
	if err != nil {
		text404(res)
		return
	}

	valid, err := db.ValidateOP(id, board)
	if err != nil {
		text500(res, req, err)
		return
	}
	if !valid {
		text404(res)
		return
	}

	counter, err := db.ThreadCounter(id)
	if err != nil {
		text500(res, req, err)
		return
	}
	if !pageEtag(res, req, etagStart(counter)) {
		return
	}

	data, err := db.GetThread(id, detectLastN(req))
	if err != nil {
		text500(res, req, err)
		return
	}

	writeJSON(res, req, false, data)
}

// Serves JSON for the "/all/" meta-board, that contains threads from all boards
func allBoardJSON(res http.ResponseWriter, req *http.Request) {
	counter, err := db.PostCounter()
	if err != nil {
		text500(res, req, err)
		return
	}
	if !pageEtag(res, req, etagStart(counter)) {
		return
	}

	data, err := db.GetAllBoard()
	if err != nil {
		text500(res, req, err)
		return
	}
	writeJSON(res, req, false, data)
}

// Serves board page JSON
func boardJSON(
	res http.ResponseWriter,
	req *http.Request,
	params map[string]string,
) {
	board := params["board"]
	if !auth.IsBoard(board) {
		text404(res)
		return
	}
	counter, err := db.BoardCounter(board)
	if err != nil {
		text500(res, req, err)
		return
	}
	if !pageEtag(res, req, etagStart(counter)) {
		return
	}
	data, err := db.GetBoard(board)
	if err != nil {
		text500(res, req, err)
		return
	}
	writeJSON(res, req, false, data)
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
	data, err := json.Marshal(list)
	serveJSON(res, req, data, err)
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

	writeJSON(res, req, true, boards)
}
