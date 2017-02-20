// Various POST request handlers

package server

import (
	"net/http"
	"strconv"

	"database/sql"

	"../auth"
	"../config"
	"../db"
	"../imager"
	"../server/websockets"
)

// Spoiler an already allocated image
func spoilerImage(w http.ResponseWriter, r *http.Request) {
	var msg spoilerRequest
	if !decodeJSON(w, r, &msg) {
		return
	}

	hash, err := db.GetPostPassword(msg.ID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		text404(w)
		return
	default:
		text500(w, r, err)
		return
	}

	if err := auth.BcryptCompare(msg.Password, hash); err != nil {
		text403(w, err)
		return
	}

	post, err := db.GetPost(msg.ID)
	switch {
	case err != nil:
		text500(w, r, err)
	case post.Image == nil:
		text400(w, errNoImage)
	case post.Image.Spoiler: // NOOP. Consider to be due to sync race.
	default:
		if err := db.SpoilerImage(msg.ID); err != nil {
			text500(w, r, err)
		}
	}
}

// Create a thread with a finished OP and immediately close it
func createThread(w http.ResponseWriter, r *http.Request) {
	maxSize := config.Get().MaxSize*1024*1024 + jsonLimit
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxSize))

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
		Captcha: f.Get("captcha"),
		ReplyCreationRequest: websockets.ReplyCreationRequest{
			Image: websockets.ImageRequest{
				Spoiler: f.Get("spoiler") == "on",
				Token:   token,
				Name:    header.Filename,
			},
			SessionCreds: auth.SessionCreds{
				UserID:  f.Get("userID"),
				Session: f.Get("session"),
			},
			Name:     f.Get("name"),
			Password: f.Get("password"),
			Body:     f.Get("body"),
		},
	}

	id, _, _, err := websockets.ConstructThread(req, auth.GetIP(r), true)
	if err != nil {

		// TODO: Not all codes are actually 400. Need to differentiate.

		text400(w, err)
		return
	}

	w.Write([]byte(strconv.FormatUint(id, 10)))
}
