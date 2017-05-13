package server

import (
	"database/sql"
	"fmt"
	"meguca/auth"
	"meguca/cache"
	"meguca/config"
	"meguca/db"
	"meguca/lang"
	"meguca/templates"
	"net/http"
	"strconv"
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
func boardHTML(
	w http.ResponseWriter,
	r *http.Request,
	p map[string]string,
	catalog bool,
) {
	b := p["board"]
	if !auth.IsBoard(b) {
		text404(w)
		return
	}
	if !assertNotBanned(w, r, b) {
		return
	}

	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}

	html, ctr, err := cache.GetHTML(chooseBoardCache(b, catalog))
	if err != nil {
		text500(w, r, err)
		return
	}

	_, hash := config.GetClient()
	etag := formatEtag(ctr, lp.ID, hash)
	if checkClientEtag(w, r, etag) {
		return
	}

	html = templates.Board(b, lp, isMinimal(r), catalog, html)
	serveHTML(w, r, etag, html, nil)
}

// Returns, if the minimal query string is not set
func isMinimal(r *http.Request) bool {
	return r.URL.Query().Get("minimal") == "true"
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

	html = templates.Thread(lp, id, p["board"], isMinimal(r), html)
	serveHTML(w, r, etag, html, nil)
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
	owned, err := db.GetOwnedBoards(userID)
	if err != nil {
		text500(w, r, err)
		return
	}

	// Retrieve titles of boards
	ownedTitles := make(config.BoardTitles, 0, len(owned))
	for _, b := range config.GetBoardTitles() {
		for _, o := range owned {
			if b.ID == o {
				ownedTitles = append(ownedTitles, b)
				break
			}
		}
	}

	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}

	serveHTML(w, r, "", []byte(templates.OwnedBoard(ownedTitles, lp.UI)), nil)
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

// Render a form for assigning staff to a board
func staffAssignmentForm(
	w http.ResponseWriter,
	r *http.Request,
	p map[string]string,
) {
	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}

	s, err := db.GetStaff(p["board"])
	if err != nil {
		text500(w, r, err)
		return
	}
	data := []byte(templates.StaffAssignment(
		[...][]string{s["owners"], s["moderators"], s["janitors"]},
		lp,
	))
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
	if !isAdmin(w, r) {
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

// Render a form with nothing but captcha and confirmation buttons
func renderCaptcha(w http.ResponseWriter, r *http.Request) {
	staticTemplate(w, r, templates.CaptchaConfirmation)
}

// Render a link to request a new captcha
func noscriptCaptchaLink(w http.ResponseWriter, r *http.Request) {
	staticTemplate(w, r, templates.NoscriptCaptchaLink)
}

// Render the captcha for noscript browsers
func noscriptCaptcha(w http.ResponseWriter, r *http.Request) {
	lp, err := lang.Get(w, r)
	if err != nil {
		text500(w, r, err)
		return
	}

	ip, err := auth.GetIP(r)
	if err != nil {
		text400(w, err)
		return
	}

	html := []byte(templates.NoscriptCaptcha(ip, lp))
	serveHTML(w, r, "", html, nil)
}

// Redirect the client to the appropriate board through a cross-board redirect
func crossRedirect(
	w http.ResponseWriter,
	r *http.Request,
	p map[string]string,
) {
	idStr := p["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		text404(w)
		return
	}

	board, op, err := db.GetPostParenthood(id)
	switch err {
	case nil:
		url := r.URL
		url.Path = fmt.Sprintf("/%s/%d", board, op)
		url.Fragment = idStr
		http.Redirect(w, r, url.String(), 301)
	case sql.ErrNoRows:
		text404(w)
	default:
		text500(w, r, err)
	}
}
