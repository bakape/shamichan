// Various POST request handlers

package server

import (
	"fmt"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"meguca/imager"
	"meguca/websockets"
	"meguca/websockets/feeds"
	"net/http"
	"strconv"
	"strings"
)

// Create a thread with a closed OP
func createThread(w http.ResponseWriter, r *http.Request) {
	repReq, ok := parsePostCreationForm(w, r)
	if !ok {
		return
	}

	// Map form data to websocket thread creation request
	f := r.Form
	req := websockets.ThreadCreationRequest{
		Subject:              f.Get("subject"),
		Board:                f.Get("board"),
		ReplyCreationRequest: repReq,
	}

	ip, err := auth.GetIP(r)
	if err != nil {
		httpError(w, r, common.StatusError{err, 400})
		return
	}
	post, err := websockets.CreateThread(req, ip)
	if err != nil {

		// TODO: Not all codes are actually 400. Need to differentiate.
		httpError(w, r, common.StatusError{err, 400})
		return
	}

	// Let the JS add the ID of the post to "mine"
	http.SetCookie(w, &http.Cookie{
		Name:  "addMine",
		Value: strconv.FormatUint(post.ID, 10),
		Path:  "/",
	})

	http.Redirect(w, r, fmt.Sprintf(`/%s/%d`, req.Board, post.ID), 303)
}

// ok = false, if failed and caller should return
func parsePostCreationForm(w http.ResponseWriter, r *http.Request) (
	req websockets.ReplyCreationRequest, ok bool,
) {
	maxSize := config.Get().MaxSize<<20 + jsonLimit
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxSize))
	err := r.ParseMultipartForm(0)
	if err != nil {
		httpError(w, r, common.StatusError{err, 400})
		return
	}

	// Handle image, if any, and extract file name
	var token string
	file, header, err := r.FormFile("image")
	if file != nil {
		file.Close()
	}
	switch err {
	case nil:
		token, err = imager.ParseUpload(r)
		if err != nil {
			imager.LogError(w, r, err)
			return
		}
	case http.ErrMissingFile:
		err = nil
	default:
		httpError(w, r, err)
		return
	}

	f := r.Form
	req = websockets.ReplyCreationRequest{
		// HTTP uses "\r\n" for newlines, but "\r" is considered non-printable
		// and raises parser.ErrContainsNonPrintable during parsing.
		Body: strings.Replace(f.Get("body"), "\r", "", -1),
		Name: f.Get("name"),
		Sage: f.Get("sage") == "on",
	}
	req.Captcha.Solution.FromRequest(r)
	if f.Get("staffTitle") == "on" {
		req.SessionCreds = extractLoginCreds(r)
	}
	if token != "" {
		req.Image = websockets.ImageRequest{
			Spoiler: f.Get("spoiler") == "on",
			Token:   token,
			Name:    header.Filename,
		}
	}

	ok = true
	return
}

// Create a closed reply post
func createReply(w http.ResponseWriter, r *http.Request) {
	req, ok := parsePostCreationForm(w, r)
	if !ok {
		return
	}

	// Validate thread
	op, err := strconv.ParseUint(r.Form.Get("op"), 10, 64)
	if err != nil {
		httpError(w, r, common.StatusError{err, 400})
		return
	}
	board := r.Form.Get("board")
	ok, err = db.ValidateOP(op, board)
	switch {
	case err != nil:
		httpError(w, r, err)
		return
	case !ok:
		httpError(w, r, common.ErrInvalidThread(op, board))
		return
	}

	ip, err := auth.GetIP(r)
	if err != nil {
		httpError(w, r, common.StatusError{err, 400})
		return
	}
	post, msg, err := websockets.CreatePost(op, board, ip, true, req)
	if err != nil {

		// TODO: Not all codes are actually 400. Need to differentiate.
		httpError(w, r, common.StatusError{err, 400})
		return
	}

	feeds.InsertPostInto(post.StandalonePost, msg)
	url := fmt.Sprintf(`/%s/%d?last100=true#bottom`, board, op)
	http.Redirect(w, r, url, 303)
}
