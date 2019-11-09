package server

import (
	"net/http"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/cache"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
)

func setHTMLHeaders(w http.ResponseWriter) {
	head := w.Header()
	for key, val := range vanillaHeaders {
		head.Set(key, val)
	}
	head.Set("Content-Type", "text/html")
}

// Serve index page HTML
func indexHTML(w http.ResponseWriter, r *http.Request) {
	board := extractParam(r, "board")
	if !auth.IsBoard(board) {
		text404(w)
		return
	}

	httpError(w, r, func() (err error) {

		pos := common.NotLoggedIn
		creds := extractLoginCreds(r)
		if creds.UserID != "" {
			var loggedIn bool
			loggedIn, err = db.IsLoggedIn(creds.UserID, creds.Session)
			switch err {
			case common.ErrInvalidCreds:
				err = nil
			case nil:
				if loggedIn {
					pos, err = db.FindPosition(board, creds.UserID)
					if err != nil {
						return
					}
				}
			default:
				return
			}
		}

		setHTMLHeaders(w)
		return cache.IndexHTML(w, r, pos)
	}())
}
