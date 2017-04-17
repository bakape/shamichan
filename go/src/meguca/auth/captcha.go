package auth

import (
	"meguca/config"
	"net/http"

	"github.com/dchest/captcha"
)

// Captcha contains the ID and solution of a captcha-protected request
type Captcha struct {
	CaptchaID, Solution string
}

var captchaServer = captcha.Server(captcha.StdWidth, captcha.StdHeight)

// NewCaptchaID creates a new captcha and write its ID to the client
func NewCaptchaID(w http.ResponseWriter, _ *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "text/plain")
	h.Set("Cache-Control", "no-store, private")
	w.Write([]byte(captcha.New()))
}

// ServeCaptcha serves captcha images and audio
func ServeCaptcha(w http.ResponseWriter, r *http.Request) {
	captchaServer.ServeHTTP(w, r)
}

// AuthenticateCaptcha posts a request to the SolveMedia API to authenticate a
// captcha
func AuthenticateCaptcha(req Captcha) bool {
	// Captchas disabled or running tests. Can not use API, when testing
	if !config.Get().Captcha {
		return true
	}
	return captcha.VerifyString(req.CaptchaID, req.Solution)
}
