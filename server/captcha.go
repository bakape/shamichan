package server

import (
	"fmt"
	"net/http"
	"time"

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

		var c auth.Captcha
		c.FromRequest(r)
		err = db.ValidateCaptcha(c, ip)
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

// Assert IP has solved a captcha
func assertSolvedCaptcha(r *http.Request) (err error) {
	ip, err := auth.GetIP(r)
	if err != nil {
		return
	}
	has, err := db.SolvedCaptchaRecently(ip, time.Minute)
	if err != nil {
		return
	}
	if !has {
		err = errInvalidCaptcha
	}
	return
}
