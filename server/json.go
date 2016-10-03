package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
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
	query := req.URL.Query().Get("last")
	if query != "" {
		lastN, err := strconv.Atoi(query)
		if err == nil && lastN <= 500 {
			return lastN
		}
	}
	return 0
}

// Serve public configuration information as JSON
func serveConfigs(w http.ResponseWriter, r *http.Request) {
	json, etag := config.GetClient()
	if checkClientEtag(w, r, etag) {
		return
	}
	setHeaders(w, etag)
	setJSONCType(w)
	writeData(w, r, json)
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

	data, err := json.Marshal(post)
	serveJSON(w, r, data, err)
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
		serveJSON(w, r, config.AllBoardConfigs, nil)
		return
	}
	if !auth.IsNonMetaBoard(board) {
		text404(w)
		return
	}

	var conf config.BoardConfigs
	if err := db.One(db.GetBoardConfig(board), &conf); err != nil {
		text500(w, r, err)
		return
	}

	data, err := conf.MarshalPublicJSON()
	serveJSON(w, r, data, err)
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
	if !pageEtag(w, r, etagStart(counter)) {
		return
	}

	data, err := db.GetThread(id, detectLastN(r))
	if err != nil {
		respondToJSONError(w, r, err)
		return
	}

	writeJSON(w, r, false, data)
}

// Serves JSON for the "/all/" meta-board, that contains threads from all boards
func allBoardJSON(w http.ResponseWriter, r *http.Request) {
	counter, err := db.PostCounter()
	if err != nil {
		text500(w, r, err)
		return
	}
	if !pageEtag(w, r, etagStart(counter)) {
		return
	}

	data, err := db.GetAllBoard()
	if err != nil {
		text500(w, r, err)
		return
	}
	writeJSON(w, r, false, data)
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
	if !pageEtag(w, r, etagStart(counter)) {
		return
	}
	data, err := db.GetBoard(board)
	if err != nil {
		text500(w, r, err)
		return
	}
	writeJSON(w, r, false, data)
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

// // Spoiler an already allocated image
// func spoilerImage(w http.ResponseWriter, req *http.Request) {
// 	var msg spoilerRequest
// 	if !decodeJSON(w, req, &msg) {
// 		return
// 	}

// 	var res struct {
// 		Image    types.Image
// 		Password []byte
// 	}
// 	q := db.FindPost(msg.ID).Pluck("image", "password").Default(nil)
// 	if err := db.One(q, &res); err != nil {
// 		text500(w, req, err)
// 		return
// 	}

// 	if res.Image.SHA1== "" {
// 		text400(w, errNoImage)
// 		return
// 	}
// 	if res.Image.Spoiler { // Already spoilered. NOOP.
// 		return
// 	}
// 	if err := auth.BcryptCompare(msg.Password, res.Password); err != nil {
// 		text403(w, err)
// 		return
// 	}

// 	logMsg, err := websockets.EncodeMessage(websockets.MessageSpoiler, msg.ID)
// 	if err != nil {
// 		text500(w, req, err)
// 		return
// 	}

// 	update := map[string]map[string]bool{
// 		"image": {
// 			"spoiler": true,
// 		},
// 	}
// 	diff := websockets.CreateUpdate(msg.ID, update, logMsg)
// 	q = db.FindParentThread(msg.ID).Update(diff)
// 	if err := db.Write(q); err != nil {
// 		text500(w, req, err)
// 		return
// 	}
// }
