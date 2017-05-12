// Various POST request handlers

package server

import (
	"fmt"
	"meguca/auth"
	"meguca/config"
	"meguca/db"
	"meguca/imager"
	"meguca/websockets"
	"meguca/websockets/feeds"
	"net/http"
	"strconv"
)

// Create a thread with a closed OP
func createThread(w http.ResponseWriter, r *http.Request) {
	repReq, ok := parsePostCreationForm(w, r)
	if !ok {
		return
	}

	// Map form data to websocket thread creation request
	req := websockets.ThreadCreationRequest{
		Subject:              r.Form.Get("subject"),
		Board:                r.Form.Get("board"),
		ReplyCreationRequest: repReq,
	}

	ip, err := auth.GetIP(r)
	if err != nil {
		text400(w, err)
		return
	}
	post, err := websockets.CreateThread(req, ip)
	if err != nil {

		// TODO: Not all codes are actually 400. Need to differentiate.

		text400(w, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf(`/%s/%d`, req.Board, post.ID), 302)
}

// ok = false, if failed and caller should return
func parsePostCreationForm(w http.ResponseWriter, r *http.Request) (
	req websockets.ReplyCreationRequest, ok bool,
) {
	maxSize := config.Get().MaxSize<<20 + jsonLimit
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxSize))
	err := r.ParseMultipartForm(0)
	if err != nil {
		text400(w, err)
		return
	}

	// Handle image, if any, and extract file name
	var token string
	_, header, err := r.FormFile("image")
	switch err {
	case nil:
		var code int
		code, token, err = imager.ParseUpload(r)
		if err != nil {
			imager.LogError(w, r, code, err)
			return
		}
	case http.ErrMissingFile:
		err = nil
	default:
		text500(w, r, err)
		return
	}

	f := r.Form
	req = websockets.ReplyCreationRequest{
		Name: f.Get("name"),
		Body: f.Get("body"),
		Sage: f.Get("sage") == "on",
		Captcha: auth.Captcha{
			CaptchaID: f.Get("captchaID"),
			Solution:  f.Get("captcha"),
		},
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
		text400(w, err)
		return
	}
	board := r.Form.Get("board")
	ok, err = db.ValidateOP(op, board)
	switch {
	case err != nil:
		text500(w, r, err)
		return
	case !ok:
		text400(w, fmt.Errorf("invalid thread: /%s/%d", board, op))
		return
	}

	ip, err := auth.GetIP(r)
	if err != nil {
		text400(w, err)
		return
	}
	post, msg, err := websockets.CreatePost(op, board, ip, true, req)
	if err != nil {

		// TODO: Not all codes are actually 400. Need to differentiate.

		text400(w, err)
		return
	}

	feeds.InsertPostInto(post.StandalonePost, msg)
	url := fmt.Sprintf(`/%s/%d?last100=true#bottom`, board, op)
	http.Redirect(w, r, url, 302)
}
