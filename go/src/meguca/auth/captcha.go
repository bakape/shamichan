package auth

import (
	"meguca/config"
	"net/http"
	"sync"
	"time"

	"github.com/dchest/captcha"
)

const captchaLifetime = time.Minute * 20

var (
	captchaServer    = captcha.Server(captcha.StdWidth, captcha.StdHeight)
	noscriptCaptchas = noscriptCaptchaMap{
		m: make(map[string]noscriptCaptcha, 64),
	}
)

// Captcha contains the ID and solution of a captcha-protected request
type Captcha struct {
	CaptchaID, Solution string
}

// Captchas for noscript users are IP-specific and need eventual cleanup
type noscriptCaptchaMap struct {
	sync.Mutex
	m map[string]noscriptCaptcha
}

type noscriptCaptcha struct {
	id      string
	created time.Time
}

// Returns a captcha id by IP. If a captcha for this IP already exists, it is
// reloaded and returned. Otherwise, a new captcha is created.
func (n *noscriptCaptchaMap) get(ip string) string {
	n.Lock()
	defer n.Unlock()

	old, ok := n.m[ip]

	// No existing captcha, it expired or this IP already used the captcha
	if !ok || !captcha.Reload(old.id) {
		id := captcha.New()
		n.m[ip] = noscriptCaptcha{
			id:      id,
			created: time.Now(),
		}
		return id
	}

	old.created = time.Now()
	n.m[ip] = old
	return old.id
}

// Remove expired ip -> captchaID mappings
func (n *noscriptCaptchaMap) cleanUp() {
	n.Lock()
	defer n.Unlock()

	till := time.Now().Add(-captchaLifetime)
	for ip, c := range n.m {
		if c.created.Before(till) {
			delete(n.m, ip)
		}
	}
}

func init() {
	captcha.SetCustomStore(captcha.NewMemoryStore(1<<10, captchaLifetime))

	go func() {
		t := time.Tick(time.Minute)
		for {
			<-t
			noscriptCaptchas.cleanUp()
		}
	}()
}

// NewCaptchaID creates a new captcha and write its ID to the client
func NewCaptchaID(w http.ResponseWriter, _ *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "text/plain")
	h.Set("Cache-Control", "no-store, private")
	w.Write([]byte(captcha.New()))
}

// GetNoscriptCaptcha returns a captcha id by IP. Use only for clients with
// scripts disabled.
func GetNoscriptCaptcha(ip string) string {
	return noscriptCaptchas.get(ip)
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
