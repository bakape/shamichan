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
