package server

import (
	"net/http"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
	"github.com/bakape/meguca/templates"
)

// Apply headers and write HTML to client
func serveHTML(
	w http.ResponseWriter,
	r *http.Request,
	data []byte,
	etag string,
) {
	head := w.Header()
	for key, val := range vanillaHeaders {
		head.Set(key, val)
	}
	head.Set("ETag", etag)
	head.Set("Content-Type", "text/html")

	writeData(w, r, data)
}

// Serves board HTML to regular or noscript clients
func boardHTML(w http.ResponseWriter, r *http.Request, p map[string]string) {
	b := p["board"]
	if !auth.IsBoard(b) {
		text404(w)
		return
	}

	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}

	_, hash := config.GetClient()
	board, etag, ok := boardData(w, r, b, lp.ID, hash)
	if !ok {
		return
	}

	data, err := templates.Board(b, lp, board)
	if err != nil {
		text500(w, r, err)
		return
	}
	serveHTML(w, r, data, etag)
}

// Asserts a thread exists on the specific board and renders the index template
func threadHTML(w http.ResponseWriter, r *http.Request, p map[string]string) {
	_, ok := validateThread(w, r, p)
	if !ok {
		return
	}

	// thread, etag, ok := threadData(w, r, id)
	// if !ok {
	// 	return
	// }
	// data, err := templates.Thread(thread)
	// if err != nil {
	// 	text500(w, r, err)
	// 	return
	// }
	// serveHTML(w, r, data, etag)
}
