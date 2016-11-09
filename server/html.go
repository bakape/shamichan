package server

import (
	"net/http"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/templates"
)

// Serves the standard HTML for desktop or mobile pages
func serveIndexTemplate(w http.ResponseWriter, r *http.Request) {
	tmpl := templates.Get("index")
	// If etags match, no need to rerender
	if checkClientEtag(w, r, tmpl.Hash) {
		return
	}
	serveHTML(w, r, tmpl.HTML, tmpl.Hash)
}

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

	if !isNoscript(r) {
		serveIndexTemplate(w, r)
		return
	}

	board, etag, ok := boardData(w, r, b)
	if !ok {
		return
	}
	data, err := templates.Board(b, board)
	if err != nil {
		text500(w, r, err)
		return
	}
	serveHTML(w, r, data, etag)
}

func isNoscript(r *http.Request) bool {
	return r.URL.Query().Get("noscript") == "true"
}

// Asserts a thread exists on the specific board and renders the index template
func threadHTML(w http.ResponseWriter, r *http.Request, p map[string]string) {
	id, ok := validateThread(w, r, p)
	if !ok {
		return
	}

	if !isNoscript(r) {
		serveIndexTemplate(w, r)
		return
	}

	thread, etag, ok := threadData(w, r, id)
	if !ok {
		return
	}
	data, err := templates.Thread(thread)
	if err != nil {
		text500(w, r, err)
		return
	}
	serveHTML(w, r, data, etag)
}
