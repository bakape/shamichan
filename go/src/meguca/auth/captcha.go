package auth

import (
	"log"
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
	failedCaptchas = failedCaptchaMap{
		m: make(map[string]failedCaptcha, 64),
	}
)

func init() {
	captcha.SetCustomStore(captcha.NewMemoryStore(1<<10, captchaLifetime))

	go func() {
		t := time.Tick(time.Minute)
		for {
			<-t
			noscriptCaptchas.cleanUp()
			failedCaptchas.cleanUp()
		}
	}()
}

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

// Stores failed captcha attempts
type failedCaptchaMap struct {
	sync.Mutex
	m map[string]failedCaptcha
}

type failedCaptcha struct {
	count      int
	firsFailed time.Time
}

// Increments the failed captcha attempts of an IP. Returns, if the user should
// be banned
func (f *failedCaptchaMap) increment(ip string) bool {
	f.Lock()
	defer f.Unlock()

	cur, ok := f.m[ip]
	cur.count++
	if cur.count == 3 {
		delete(f.m, ip)
		return true
	}
	if !ok {
		cur.firsFailed = time.Now()
	}
	f.m[ip] = cur
	return false
}

// Remove entries older than 20 minutes
func (f *failedCaptchaMap) cleanUp() {
	f.Lock()
	defer f.Unlock()

	till := time.Now().Add(-20 * time.Minute)
	for ip, fc := range f.m {
		if fc.firsFailed.Before(till) {
			delete(f.m, ip)
		}
	}
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
// captcha.
// SystemBan forwards db.SystemBan to avoid circular imports
func AuthenticateCaptcha(req Captcha, ip string,
	systemBan func(string, string, time.Time) error,
) bool {
	// Captchas disabled or running tests. Can not use API, when testing
	if !config.Get().Captcha {
		return true
	}
	passed := captcha.VerifyString(req.CaptchaID, req.Solution)
	if !passed && failedCaptchas.increment(ip) {
		err := systemBan(ip, "bot detected", time.Now().Add(time.Hour*48))
		if err != nil {
			log.Printf("automatic ban: %s\n", err)
		}
	}
	return passed
}
