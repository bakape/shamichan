// Various POST request handlers

package server

import (
	"net/http"
	"strconv"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager"
	"github.com/bakape/meguca/server/websockets"
	"github.com/bakape/meguca/types"
)

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

// Create a thread with a finished OP and immediately close it
func createThread(w http.ResponseWriter, r *http.Request) {
	maxSize := config.Get().MaxSize*1024*1024 + jsonLimit
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)
	w.Header().Set("Access-Control-Allow-Origin", config.AllowedOrigin)

	code, token, err := imager.ParseUpload(r)
	if err != nil {
		imager.LogError(w, r, code, err)
		return
	}

	// Extract file name
	_, header, err := r.FormFile("image")
	if err != nil {
		text500(w, r, err)
		return
	}

	// Map form data to websocket thread creation request
	f := r.Form
	req := websockets.ThreadCreationRequest{
		Subject: f.Get("subject"),
		Board:   f.Get("board"),
		Captcha: types.Captcha{
			Captcha:   f.Get("captcha"),
			CaptchaID: "manual_challenge",
		},
		ReplyCreationRequest: websockets.ReplyCreationRequest{
			Image: websockets.ImageRequest{
				Spoiler: f.Get("spoiler") == "on",
				Token:   token,
				Name:    header.Filename,
			},
			Name:     f.Get("name"),
			Email:    f.Get("email"),
			Auth:     f.Get("auth"),
			Password: f.Get("password"),
			Body:     f.Get("body"),
		},
	}

	id, _, err := websockets.ConstructThread(req, auth.GetIP(r), true)
	if err != nil {
		// TODO: Not all codes are actually 400. Need to differentiate.
		text400(w, err)
		return
	}

	w.Write(strconv.AppendInt(nil, id, 10))
}
