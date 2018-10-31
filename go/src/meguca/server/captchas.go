package server

import (
	"fmt"
	"meguca/auth"
	"meguca/common"
	"meguca/db"
	"meguca/templates"
	"net/http"
)

// Signifies captcha service is not yet loaded
type errCaptchasNotReady string

func (e errCaptchasNotReady) Error() string {
	return fmt.Sprintf("captchas not initialized for board %s", string(e))
}

// Authenticate a captcha solution
func authenticateCaptcha(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		if !assertNotBanned(w, r, "all") {
			return
		}
		err = r.ParseForm()
		if err != nil {
			return common.StatusError{err, 400}
		}
		ip, err := auth.GetIP(r)
		if err != nil {
			return
		}
		var c auth.Captcha
		c.FromRequest(r)
		err = db.ValidateCaptcha(c, ip)
		switch err {
		case nil:
		case common.ErrInvalidCaptcha:
			http.Redirect(w, r, r.RemoteAddr, 302)
			return nil
		default:
			return
		}

		w.Write([]byte(r.Form.Get("captchouli-id")))
		err = db.ResetSpamScore(ip)
		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Create new captcha and write its HTML to w. Colour and background can be left
// blank to use defaults.
func serveNewCaptcha(w http.ResponseWriter, r *http.Request) {
	b := extractParam(r, "board")
	if !assertNotBanned(w, r, "all") {
		return
	}
	s := auth.CaptchaService(b)
	if s == nil {
		httpError(w, r, errCaptchasNotReady(b))
		return
	}
	s.ServeNewCaptcha(w, r)
}

// Render a form with nothing but captcha and confirmation buttons
func renderCaptchaConfirmation(w http.ResponseWriter, r *http.Request) {
	setHTMLHeaders(w)
	templates.WriteCaptchaConfirmation(w)
}

// Render a link to request a new captcha
func noscriptCaptchaLink(w http.ResponseWriter, r *http.Request) {
	setHTMLHeaders(w)
	templates.WriteNoscriptCaptchaLink(w, extractParam(r, "board"))
}
