package server

import (
	"net/http"
	"sort"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/cache"
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

	html, ctr, err := cache.GetHTML(cache.BoardKey(b), boardCache)
	if err != nil {
		text500(w, r, err)
		return
	}

	_, hash := config.GetClient()
	etag := formatEtag(ctr, lp.ID, hash)
	if checkClientEtag(w, r, etag) {
		return
	}

	html, err = templates.Board(b, lp, withIndex(r), html)
	serveHTML(w, r, etag, html, err)
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

	k := cache.ThreadKey(id, detectLastN(r))
	html, ctr, err := cache.GetHTML(k, threadCache)
	if err != nil {
		respondToJSONError(w, r, err)
		return
	}

	_, hash := config.GetClient()
	etag := formatEtag(ctr, lp.ID, hash)
	if checkClientEtag(w, r, etag) {
		return
	}

	html, err = templates.Thread(lp, withIndex(r), html)
	serveHTML(w, r, etag, html, err)
}

// Render a board selection and navigation panel and write HTML to client
func boardNavigation(w http.ResponseWriter, r *http.Request) {
	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}
	serveHTML(w, r, "", []byte(templates.BoardNavigation(lp.UI)), nil)
}

// Execute a simple template, that only accepts a language pack argument
func staticTemplate(
	w http.ResponseWriter,
	r *http.Request,
	fn func(lang.Pack) string,
) {
	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}
	serveHTML(w, r, "", []byte(fn(lp)), nil)
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

	serveHTML(w, r, "", []byte(templates.OwnedBoard(owned, lp.UI)), nil)
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

	data := []byte(templates.ConfigureBoard(conf, lp))
	serveHTML(w, r, "", data, nil)
}

// Renders a form for creating new boards
func boardCreationForm(w http.ResponseWriter, r *http.Request) {
	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}
	serveHTML(w, r, "", []byte(templates.CreateBoard(lp)), nil)
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

	data := []byte(templates.ConfigureServer((*config.Get()), lp))
	serveHTML(w, r, "", data, nil)
}

// Render a form to change an account password
func changePasswordForm(w http.ResponseWriter, r *http.Request) {
	staticTemplate(w, r, templates.ChangePassword)
}
