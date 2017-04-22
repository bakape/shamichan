// Various POST request handlers

package server

import (
	"fmt"
	"meguca/auth"
	"meguca/config"
	"meguca/imager"
	"meguca/websockets"
	"net/http"
)

// Create a thread with a finished OP and immediately close it
func createThread(w http.ResponseWriter, r *http.Request) {
	maxSize := config.Get().MaxSize*1024*1024 + jsonLimit
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

	// Map form data to websocket thread creation request
	f := r.Form
	req := websockets.ThreadCreationRequest{
		Subject: f.Get("subject"),
		Board:   f.Get("board"),
		Captcha: auth.Captcha{
			CaptchaID: f.Get("captchaID"),
			Solution:  f.Get("captcha"),
		},
		ReplyCreationRequest: websockets.ReplyCreationRequest{
			Name: f.Get("name"),
			Body: f.Get("body"),
		},
	}
	if token != "" {
		req.Image = websockets.ImageRequest{
			Spoiler: f.Get("spoiler") == "on",
			Token:   token,
			Name:    header.Filename,
		}
	}

	id, _, _, err := websockets.ConstructThread(req, auth.GetIP(r), true)
	if err != nil {

		// TODO: Not all codes are actually 400. Need to differentiate.

		text400(w, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf(`/%s/%d`, req.Board, id), 302)
}
