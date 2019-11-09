package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bakape/meguca/config"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/templates"
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

		var (
			c       auth.Captcha
			session auth.Base64Token
		)
		c.FromRequest(r)
		err = session.EnsureCookie(w, r)
		if err != nil {
			return
		}
		err = db.ValidateCaptcha(c, session, ip)
		if err == common.ErrInvalidCaptcha {
			b := extractParam(r, "board")
			s := auth.CaptchaService(b)
			if s == nil {
				return errCaptchasNotReady(b)
			}
			return s.ServeNewCaptcha(w, r)
		}
		if err != nil {
			return
		}

		w.Write([]byte("OK"))
		return
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Create new captcha and write its HTML to w. Colour and background can be left
// blank to use defaults.
func serveNewCaptcha(w http.ResponseWriter, r *http.Request) {
	httpError(w, r, func() (err error) {
		b := extractParam(r, "board")
		if !assertNotBanned(w, r, "all") {
			return
		}

		ip, err := auth.GetIP(r)
		if err != nil {
			return
		}
		var session auth.Base64Token
		err = session.EnsureCookie(w, r)
		if err != nil {
			return
		}
		db.IncrementSpamScore(session, ip, config.Get().ImageScore)

		s := auth.CaptchaService(b)
		if s == nil {
			return errCaptchasNotReady(b)
		}
		s.ServeNewCaptcha(w, r)
		return
	}())
}

// Render a form with nothing but captcha and confirmation buttons
func renderCaptchaConfirmation(w http.ResponseWriter, r *http.Request) {
	setHTMLHeaders(w)
	templates.WriteCaptchaConfirmation(w)
}

// Assert IP has solved a captcha
func assertSolvedCaptcha(w http.ResponseWriter, r *http.Request) (err error) {
	var session auth.Base64Token
	err = session.EnsureCookie(w, r)
	if err != nil {
		return
	}
	has, err := db.SolvedCaptchaRecently(session, time.Minute)
	if err != nil {
		return
	}
	if !has {
		err = errInvalidCaptcha
	}
	return
}
