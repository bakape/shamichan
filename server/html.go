package server

import (
	"net/http"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/cache"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/templates"
)

func setHTMLHeaders(w http.ResponseWriter) {
	head := w.Header()
	for key, val := range vanillaHeaders {
		head.Set(key, val)
	}
	head.Set("Content-Type", "text/html")
}

// Serves board HTML to regular or noscript clients
func boardHTML(w http.ResponseWriter, r *http.Request, b string, catalog bool) {
	if !auth.IsBoard(b) {
		text404(w)
		return
	}
	if !assertNotBanned(w, r, b) {
		return
	}

	theme := resolveTheme(r, b)
	html, data, ctr, err := cache.GetHTML(boardCacheArgs(r, b, catalog))
	switch err {
	case nil:
	case cache.ErrPageOverflow:
		text404(w)
		return
	default:
		httpError(w, r, err)
		return
	}

	pos, ok := extractPosition(w, r)
	if !ok {
		return
	}

	_, hash := config.GetClient()
	etag := formatEtag(ctr, hash, pos)
	if checkClientEtag(w, r, etag) {
		return
	}

	var n, total int
	if !catalog {
		p := data.(cache.PageStore)
		n = p.PageNumber
		total = p.Data.Pages
	}

	setHTMLHeaders(w)
	templates.Board(
		w,
		b, theme,
		n, total,
		pos,
		r.URL.Query().Get("minimal") == "true", catalog,
		html,
	)
}

// Resolve theme to render in accordance to client and board settings.
// Needed to prevent Flash Of Unstyled Content.
func resolveTheme(r *http.Request, board string) string {
	if c, err := r.Cookie("theme"); err == nil {
		for _, t := range common.Themes {
			if c.Value == t {
				return c.Value
			}
		}
	}
	if board == "all" {
		return config.Get().DefaultCSS
	}
	return config.GetBoardConfigs(board).DefaultCSS
}

// Asserts a thread exists on the specific board and renders the index template
func threadHTML(w http.ResponseWriter, r *http.Request) {
	id, ok := validateThread(w, r)
	if !ok {
		return
	}

	b := extractParam(r, "board")
	theme := resolveTheme(r, b)
	lastN := detectLastN(r)
	k := cache.ThreadKey(id, lastN)
	html, data, ctr, err := cache.GetHTML(k, cache.ThreadFE)
	if err != nil {
		httpError(w, r, err)
		return
	}

	pos, ok := extractPosition(w, r)
	if !ok {
		return
	}

	_, hash := config.GetClient()
	etag := formatEtag(ctr, hash, pos)
	if checkClientEtag(w, r, etag) {
		return
	}

	thread := data.(common.Thread)
	setHTMLHeaders(w)
	templates.Thread(
		w,
		id,
		b, thread.Subject, theme,
		lastN != 0, thread.Locked,
		pos,
		html,
	)
}

// Extract logged in position for HTML request.
// If ok == false, caller should return.
func extractPosition(w http.ResponseWriter, r *http.Request) (
	pos common.ModerationLevel, ok bool,
) {
	ok = true
	pos = common.NotLoggedIn
	creds := extractLoginCreds(r)
	if creds.UserID == "" {
		return
	}

	loggedIn, err := db.IsLoggedIn(creds.UserID, creds.Session)
	switch err {
	case common.ErrInvalidCreds:
		return
	case nil:
		if loggedIn {
			board := extractParam(r, "board")
			pos, err = db.FindPosition(board, creds.UserID)
			if err != nil {
				httpError(w, r, err)
				ok = false
				return
			}
		}
	default:
		httpError(w, r, err)
		ok = false
		return
	}

	return
}

// Render a board selection and navigation panel and write HTML to client
func boardNavigation(w http.ResponseWriter, r *http.Request) {
	setHTMLHeaders(w)
	templates.WriteBoardNavigation(w)
}

// Serve a form for selecting one of several boards owned by the user
func ownedBoardSelection(w http.ResponseWriter, r *http.Request) {
	userID := extractParam(r, "userID")
	owned, err := db.GetOwnedBoards(userID)
	if err != nil {
		httpError(w, r, err)
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

	setHTMLHeaders(w)
	templates.WriteOwnedBoard(w, ownedTitles)
}

// Renders a form for configuring a board owned by the user
func boardConfigurationForm(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		conf, err := boardConfData(w, r)
		if err != nil {
			return
		}

		setHTMLHeaders(w)
		templates.ConfigureBoard(w, conf)
		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Render a form for assigning staff to a board
func staffAssignmentForm(w http.ResponseWriter, r *http.Request) {
	s, err := db.GetStaff(extractParam(r, "board"))
	if err != nil {
		httpError(w, r, err)
		return
	}
	setHTMLHeaders(w)
	templates.StaffAssignment(w,
		[...][]string{s[common.BoardOwner], s[common.Moderator],
			s[common.Janitor]})
}

// Renders a form for creating new boards
func boardCreationForm(w http.ResponseWriter, r *http.Request) {
	setHTMLHeaders(w)
	templates.WriteCreateBoard(w)
}

// Render the form for configuring the server
func serverConfigurationForm(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		err = isAdmin(w, r)
		if err != nil {
			return
		}

		setHTMLHeaders(w)
		templates.ConfigureServer(w, (*config.Get()))
		return

	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Render a form to change an account password
func changePasswordForm(w http.ResponseWriter, r *http.Request) {
	setHTMLHeaders(w)
	templates.ChangePassword(w)
}

func bannerSettingForm(w http.ResponseWriter, r *http.Request) {
	setHTMLHeaders(w)
	templates.WriteBannerForm(w)
}

func loadingAnimationForm(w http.ResponseWriter, r *http.Request) {
	setHTMLHeaders(w)
	templates.WriteLoadingAnimationForm(w)
}
