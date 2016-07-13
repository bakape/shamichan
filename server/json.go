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

// Serve JSON retrieved from the database and handle i ETag-related
// functionality
func serveJSON(
	res http.ResponseWriter,
	req *http.Request,
	data []byte,
	err error,
) {
	if err != nil {
		textErrorPage(res, req, err)
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
func writeJSON(res http.ResponseWriter, req *http.Request, data interface{}) {
	JSON, err := json.Marshal(data)
	if err != nil {
		textErrorPage(res, req, err)
		return
	}
	setJSONCType(res)
	writeData(res, req, JSON)
}

func setJSONCType(res http.ResponseWriter) {
	res.Header().Set("Content-Type", "application/json")
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
		text404(res, req)
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
		text404(res, req)
	} else {
		textErrorPage(res, req, err)
	}
}

// Serve board-specific configuration JSON
func serveBoardConfigs(
	res http.ResponseWriter,
	req *http.Request,
	params map[string]string,
) {
	board := params["board"]
	if !auth.IsNonMetaBoard(board) {
		text404(res, req)
		return
	}

	var conf config.BoardConfigs
	if err := db.One(db.GetBoardConfig(board), &conf); err != nil {
		textErrorPage(res, req, err)
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
		text404(res, req)
		return
	}

	valid, err := db.ValidateOP(id, board)
	if err != nil {
		textErrorPage(res, req, err)
		return
	}
	if !valid {
		text404(res, req)
		return
	}

	counter, err := db.ThreadCounter(id)
	if err != nil {
		textErrorPage(res, req, err)
		return
	}
	if !pageEtag(res, req, etagStart(counter)) {
		return
	}

	data, err := db.GetThread(id, detectLastN(req))
	if err != nil {
		textErrorPage(res, req, err)
		return
	}

	writeJSON(res, req, data)
}

// Serves JSON for the "/all/" meta-board, that contains threads from all boards
func allBoardJSON(res http.ResponseWriter, req *http.Request) {
	counter, err := db.PostCounter()
	if err != nil {
		textErrorPage(res, req, err)
		return
	}
	if !pageEtag(res, req, etagStart(counter)) {
		return
	}

	data, err := db.GetAllBoard()
	if err != nil {
		textErrorPage(res, req, err)
		return
	}
	writeJSON(res, req, data)
}

// Serves board page JSON
func boardJSON(
	res http.ResponseWriter,
	req *http.Request,
	params map[string]string,
) {
	board := params["board"]
	if !auth.IsBoard(board) {
		text404(res, req)
		return
	}
	counter, err := db.BoardCounter(board)
	if err != nil {
		textErrorPage(res, req, err)
		return
	}
	if !pageEtag(res, req, etagStart(counter)) {
		return
	}
	data, err := db.GetBoard(board)
	if err != nil {
		textErrorPage(res, req, err)
		return
	}
	writeJSON(res, req, data)
}

// Serve a JSON array of all available boards and their titles
func serveBoardList(res http.ResponseWriter, req *http.Request) {
	var list []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	q := r.Table("boards").Pluck("id", "title")
	if err := db.All(q, &list); err != nil {
		textErrorPage(res, req, err)
		return
	}
	data, err := json.Marshal(list)
	serveJSON(res, req, data, err)
}
