package server

import (
	"net/http"

	"sort"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
	"github.com/bakape/meguca/templates"
)

// Apply headers and write HTML to client
func serveHTML(
	w http.ResponseWriter,
	r *http.Request,
	etag string,
	data []byte,
	err error,
) {
	if err != nil {
		text500(w, r, err)
		return
	}
	head := w.Header()
	for key, val := range vanillaHeaders {
		head.Set(key, val)
	}
	if etag != "" {
		head.Set("ETag", etag)
	}
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

	withIndex := r.URL.Query().Get("noIndex") != "true"
	data, err := templates.Board(b, lp, withIndex, board)
	serveHTML(w, r, etag, data, err)
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

// Render a board selection and navigation panel and write HTML to client
func boardNavigation(w http.ResponseWriter, r *http.Request) {
	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}
	data, err := templates.BoardNavigation(lp)
	serveHTML(w, r, "", data, err)
}

// Serve a form for selecting one of several boards owned by the user
func ownedBoardSelection(
	w http.ResponseWriter,
	r *http.Request,
	p map[string]string,
) {
	userID := p["userID"]

	var owned config.BoardTitles
	for _, c := range config.GetAllBoardConfigs() {
		for _, o := range c.Staff["owners"] {
			if o == userID {
				owned = append(owned, config.BoardTitle{
					ID:    c.ID,
					Title: c.Title,
				})
				break
			}
		}
	}
	sort.Sort(owned)

	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}

	data, err := templates.OwnedBoard(owned, lp)
	serveHTML(w, r, "", data, err)
}

// Renders a form for configuring a board owned by the user
func boardConfigurationForm(w http.ResponseWriter, r *http.Request) {
	conf, isValid := boardConfData(w, r)
	if !isValid {
		return
	}

	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}

	data, err := templates.ConfigureBoard(conf, lp)
	serveHTML(w, r, "", data, err)
}

// Renders a form for creating new boards
func boardCreationForm(w http.ResponseWriter, r *http.Request) {
	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}

	data, err := templates.CreateBoard(lp)
	serveHTML(w, r, "", data, err)
}
