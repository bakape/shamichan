package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/bakape/captchouli"
	captchouli_common "github.com/bakape/captchouli/common"
	"github.com/Chiiruno/meguca/common"
	"github.com/Chiiruno/meguca/config"
)

const (
	// Name of cookie that holds the captcha session
	CaptchaCookie = "captcha_session"
)

var (
	openMu           sync.Mutex
	open             bool
	servicesMu       sync.RWMutex
	globalService    *captchouli.Service
	overrideServices map[string]*captchouli.Service

	ErrInvalidToken = common.ErrInvalidInput("invalid token")
)

// 64 byte token that JSON/text en/decodes to a raw URL-safe encoding base64
// string
type Base64Token [64]byte

func (b Base64Token) MarshalText() ([]byte, error) {
	buf := make([]byte, 86)
	base64.RawURLEncoding.Encode(buf[:], b[:])
	return buf, nil
}

func (b *Base64Token) UnmarshalText(buf []byte) error {
	if len(buf) != 86 {
		return ErrInvalidToken
	}

	n, err := base64.RawURLEncoding.Decode(b[:], buf)
	if n != 64 || err != nil {
		return ErrInvalidToken
	}
	return nil
}

// Ensure client has a "captcha_session" cookie.
// If yes, read it into b.
// If not, generate new one and set it on the client.
//
// For less disruptive toggling of captchas on and off, best always ensure
// this cookie exists on the client.
func (b *Base64Token) EnsureCookie(
	w http.ResponseWriter,
	r *http.Request,
) (err error) {
	c, err := r.Cookie(CaptchaCookie)
	switch err {
	case nil:
		return b.UnmarshalText([]byte(c.Value))
	case http.ErrNoCookie:
		*b, err = NewBase64Token()
		if err != nil {
			return
		}

		var text []byte
		text, err = b.MarshalText()
		if err != nil {
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:    CaptchaCookie,
			Value:   string(text),
			Path:    "/",
			Expires: time.Now().Add(time.Hour * 24),
		})
		return
	default:
		return fmt.Errorf("auth: reading cookie: %s", err)
	}
}

// Create new Base64Token populated by cryptographically secure random data
func NewBase64Token() (b Base64Token, err error) {
	n, err := rand.Read(b[:])
	if err == nil && n != 64 {
		err = fmt.Errorf("auth: not enough data read: %d", n)
	}
	return
}

// 64 byte ID that JSON en/decodes to a base64 string
type CaptchaID [64]byte

func (b CaptchaID) MarshalJSON() ([]byte, error) {
	return json.Marshal(b[:])
}

func (b *CaptchaID) UnmarshalJSON(buf []byte) (err error) {
	var m string
	err = json.Unmarshal(buf, &m)
	if err != nil {
		return
	}
	if m != "" {
		*b, err = captchouli.DecodeID(m)
	} else {
		*b = [64]byte{}
	}
	return
}

func (b *CaptchaID) FromRequest(r *http.Request) {
	*b, _ = captchouli.ExtractID(r)
}

// Solution to a captcha with special JSON en/decoding
type CaptchaSolution []byte

func (s CaptchaSolution) MarshalJSON() ([]byte, error) {
	// []byte is automatically marshalled to string
	m := make([]uint16, len(s))
	for i, b := range s {
		m[i] = uint16(b)
	}
	return json.Marshal(m)
}

func (s *CaptchaSolution) UnmarshalJSON(buf []byte) (err error) {
	var m []uint16
	err = json.Unmarshal(buf, &m)
	if err != nil {
		return
	}
	if len(m) != 0 {
		*s = make(CaptchaSolution, len(m))
		for i, b := range m {
			(*s)[i] = byte(b)
		}
	} else {
		*s = nil
	}
	return
}

// Captcha contains the ID and solution of a captcha-protected request
type Captcha struct {
	CaptchaID CaptchaID       `json:"id"`
	Solution  CaptchaSolution `json:"solution"`
}

// Zeroes c on no captcha in request
// It is up to the caller to decide, if the returned error should or should not
// be ignored.
func (c *Captcha) FromRequest(r *http.Request) {
	c.CaptchaID, _ = captchouli.ExtractID(r)
	c.Solution, _ = captchouli.ExtractSolution(r)
}

// Retrieve captcha service for specific board
func CaptchaService(board string) *captchouli.Service {
	servicesMu.RLock()
	defer servicesMu.RUnlock()

	if globalService == nil { // Not initialized yet
		return nil
	}

	s := overrideServices[board]
	if s == nil {
		s = globalService
	}
	return s
}

// Initialize captcha services, if not already running, and launch a service for
// the configured tags with optional additional services for select boards.
// This function blocks until all services are initialized.
func LoadCaptchaServices() (err error) {
	conf := config.Get()
	if !conf.Captcha || config.Server.ImagerMode == config.NoImager {
		return
	}

	openMu.Lock()
	defer openMu.Unlock()
	if !open {
		captchouli_common.IsTest = common.IsTest
		err = captchouli.Open()
		if err != nil {
			return
		}
		open = true
	}

	opts := captchouli.Options{
		Quiet: true,
		Tags:  conf.CaptchaTags,
	}
	setRatings := func(board string) {
		if config.GetBoardConfigs(board).NSFW {
			opts.Explicitness = []captchouli.Rating{captchouli.Safe,
				captchouli.Questionable, captchouli.Explicit}
		} else {
			opts.Explicitness = nil
		}
	}

	setRatings("all")
	g, err := captchouli.NewService(opts)
	if err != nil {
		return
	}
	over := make(map[string]*captchouli.Service, len(conf.OverrideCaptchaTags))
	for b, tags := range conf.OverrideCaptchaTags {
		opts.Tags = []string{tags}
		setRatings(b)
		over[b], err = captchouli.NewService(opts)
		if err != nil {
			return
		}
	}

	servicesMu.Lock()
	defer servicesMu.Unlock()
	globalService = g
	overrideServices = over

	return
}

// Create a sample captcha for testing purposes and return it with its solution
func CreateTestCaptcha() (c Captcha, err error) {
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	err = CaptchaService("all").ServeNewCaptcha(w, r)
	if err != nil {
		return
	}
	c.CaptchaID, c.Solution, err = captchouli.ExtractCaptcha(w.Body)
	return
}
