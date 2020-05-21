package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/bakape/captchouli/v2"
	captchouli_common "github.com/bakape/captchouli/v2/common"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
)

const (
	// Name of cookie that holds the captcha session
	CaptchaCookie = "captcha_session"
)

var (
	openMu        sync.Mutex
	open          bool
	servicesMu    sync.RWMutex
	globalService *captchouli.Service

	ErrInvalidToken = common.ErrInvalidInput("invalid token")
)

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

// Retrieve captcha service. Could be nil, if service not instantiated yet.
func CaptchaService(board string) *captchouli.Service {
	servicesMu.RLock()
	defer servicesMu.RUnlock()
	return globalService
}

// Initialize captcha services, if not already running, and launch a service for
// the configured tags with optional additional services for select boards.
// This function blocks until all services are initialized.
func LoadCaptchaServices() (err error) {
	conf := config.Get()
	if !conf.Captcha {
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

	g, err := captchouli.NewService(opts)
	if err != nil {
		return
	}

	servicesMu.Lock()
	globalService = g
	servicesMu.Unlock()

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
