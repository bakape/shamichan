package auth

import (
	"encoding/json"
	"github.com/bakape/meguca/config"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/bakape/captchouli"
)

var (
	openMu           sync.Mutex
	open             bool
	servicesMu       sync.RWMutex
	globalService    *captchouli.Service
	overrideServices map[string]*captchouli.Service
)

// 64 byte ID that JSON en/decodes to a base64 string
type Base64ID [64]byte

func (b Base64ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(b[:])
}

func (b *Base64ID) UnmarshalJSON(buf []byte) (err error) {
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

func (b *Base64ID) FromRequest(r *http.Request) {
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

func (s *CaptchaSolution) FromRequest(r *http.Request) {
	*s, _ = captchouli.ExtractSolution(r)
}

// Captcha contains the ID and solution of a captcha-protected request
type Captcha struct {
	CaptchaID Base64ID
	Solution  CaptchaSolution
}

// Zeroes c on no captcha in request
func (c *Captcha) FromRequest(r *http.Request) {
	c.CaptchaID.FromRequest(r)
	c.Solution.FromRequest(r)
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
	if !conf.Captcha || config.ImagerMode == config.NoImager {
		return
	}

	openMu.Lock()
	defer openMu.Unlock()
	if !open {
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
