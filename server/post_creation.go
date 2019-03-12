// Various POST request handlers

package server

import (
	"fmt"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager"
	"github.com/bakape/meguca/websockets"
	"github.com/bakape/meguca/websockets/feeds"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Create a thread with a closed OP
func createThread(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		repReq, ip, err := parsePostCreationForm(w, r)
		if err != nil {
			return
		}

		// Map form data to websocket thread creation request
		f := r.Form
		req := websockets.ThreadCreationRequest{
			Subject:              f.Get("subject"),
			Board:                f.Get("board"),
			ReplyCreationRequest: repReq,
		}

		post, err := websockets.CreateThread(req, ip)
		if err != nil {
			// TODO: Not all codes are actually 400. Need to differentiate
			return common.StatusError{err, 400}
		}

		// Let the JS add the ID of the post to "mine"
		http.SetCookie(w, &http.Cookie{
			Name:  "addMine",
			Value: strconv.FormatUint(post.ID, 10),
			Path:  "/",
		})

		http.Redirect(w, r, fmt.Sprintf(`/%s/%d`, req.Board, post.ID), 303)
		incrementSpamscore(ip, req.Body, true)

		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// ok = false, if failed and caller should return
func parsePostCreationForm(w http.ResponseWriter, r *http.Request,
) (req websockets.ReplyCreationRequest, ip string, err error) {
	conf := config.Get()
	maxSize := conf.MaxSize<<20 + jsonLimit
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxSize))
	err = r.ParseMultipartForm(0)
	if err != nil {
		return
	}

	ip, err = auth.GetIP(r)
	if err != nil {
		err = common.StatusError{err, 400}
		return
	}

	if conf.Captcha {
		var need, has bool
		need, err = db.NeedCaptcha(ip)
		if err != nil {
			return
		}
		if need {
			err = common.ErrInvalidCaptcha
			return
		}
		has, err = db.SolvedCaptchaRecently(ip, 3*time.Minute)
		if err != nil {
			return
		}
		if !has {
			err = common.ErrInvalidCaptcha
			return
		}
	}

	f := r.Form
	req = websockets.ReplyCreationRequest{
		// HTTP uses "\r\n" for newlines, but "\r" is considered non-printable
		// and raises parser.ErrContainsNonPrintable during parsing.
		Body: strings.Replace(f.Get("body"), "\r", "", -1),
		Name: f.Get("name"),
		Sage: f.Get("sage") == "on",
	}
	if f.Get("staffTitle") == "on" {
		req.SessionCreds = extractLoginCreds(r)
	}

	// Handle image, if any, and extract file name
	var token string
	_, header, err := r.FormFile("image")
	switch err {
	case nil:
		token, err = imager.ParseUpload(r)
		if err != nil {
			return
		}
	case http.ErrMissingFile:
		err = nil
	default:
		return
	}

	if token != "" {
		req.Image = websockets.ImageRequest{
			Spoiler: f.Get("spoiler") == "on",
			Token:   token,
			Name:    header.Filename,
		}
	}

	return
}

// Create a closed reply post
func createReply(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		req, ip, err := parsePostCreationForm(w, r)
		if err != nil {
			return
		}

		// Validate thread
		op, err := strconv.ParseUint(r.Form.Get("op"), 10, 64)
		if err != nil {
			return common.StatusError{err, 400}
		}
		board := r.Form.Get("board")
		ok, err := db.ValidateOP(op, board)
		switch {
		case err != nil:
			return
		case !ok:
			return common.ErrInvalidThread(op, board)
		}

		post, msg, err := websockets.CreatePost(op, board, ip, req)
		if err != nil {
			// TODO: Not all codes are actually 400. Need to differentiate
			return common.StatusError{err, 400}
		}

		feeds.InsertPostInto(post.StandalonePost, msg)
		http.Redirect(w, r,
			fmt.Sprintf(`/%s/%d?last100=true#bottom`, board, op), 303)
		incrementSpamscore(ip, req.Body, false)

		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

func incrementSpamscore(ip, body string, isOP bool) {
	conf := config.Get()
	s := conf.CharScore * uint(utf8.RuneCountInString(body))
	s += conf.PostCreationScore
	if isOP {
		s += conf.PostCreationScore * 2
	}
	db.IncrementSpamScore(ip, time.Duration(s))
}
