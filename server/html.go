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

	data, err := templates.Board(b, lp, withIndex(r), board)
	serveHTML(w, r, etag, data, err)
}

// Returns, if the noIndex query string is not set
func withIndex(r *http.Request) bool {
	return r.URL.Query().Get("noIndex") != "true"
}

// Asserts a thread exists on the specific board and renders the index template
func threadHTML(w http.ResponseWriter, r *http.Request, p map[string]string) {
	id, ok := validateThread(w, r, p)
	if !ok {
		return
	}

	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}

	_, hash := config.GetClient()
	thread, etag, ok := threadData(w, r, id, lp.ID, hash)
	if !ok {
		return
	}

	data, err := templates.Thread(lp, withIndex(r), thread)
	serveHTML(w, r, etag, data, err)
}

// Render a board selection and navigation panel and write HTML to client
func boardNavigation(w http.ResponseWriter, r *http.Request) {
	staticTemplate(w, r, templates.BoardNavigation)
}

// Execute a simple template, that only accepts a language pack argument
func staticTemplate(
	w http.ResponseWriter,
	r *http.Request,
	fn func(lang.Pack) ([]byte, error),
) {
	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}
	data, err := fn(lp)
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
	staticTemplate(w, r, templates.CreateBoard)
}

// Render the form for configuring the server
func serverConfigurationForm(w http.ResponseWriter, r *http.Request) {
	var msg sessionCreds
	if !decodeJSON(w, r, &msg) || !isAdmin(w, r, msg) {
		return
	}

	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}

	data, err := templates.ConfigureServer((*config.Get()), lp)
	serveHTML(w, r, "", data, err)
}

// Render a form to change an account password
func changePasswordForm(w http.ResponseWriter, r *http.Request) {
	staticTemplate(w, r, templates.ChangePassword)
}
