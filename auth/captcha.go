package auth

import (
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/bakape/captchouli"
	captchouli_common "github.com/bakape/captchouli/common"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
)

var (
	openMu           sync.Mutex
	open             bool
	servicesMu       sync.RWMutex
	globalService    *captchouli.Service
	overrideServices map[string]*captchouli.Service

	ErrInvalidToken = common.ErrInvalidInput("invalid token")
)

// 64 byte token that JSON/text en/decodes to a base64 string
type Base64Token [64]byte

func (b *Base64Token) MarshalText() ([]byte, error) {
	buf := make([]byte, 86+2)
	buf[0] = '"'
	buf[86+2] = '"'
	base64.RawURLEncoding.Encode(buf[1:], b[:])
	return buf, nil
}

func (b *Base64Token) UnmarshalText(buf []byte) error {
	if len(buf) != 86 {
		return ErrInvalidToken
	}

	n, err := base64.RawStdEncoding.Decode(b[:], buf)
	if n != 64 || err != nil {
		return ErrInvalidToken
	}
	return nil
}

func (b *Base64Token) Value() (driver.Value, error) {
	buf := make([]byte, 64)
	copy(buf, b[:])
	return buf, nil
}

func (b *Base64Token) Scan(src interface{}) error {
	switch src.(type) {
	case []byte:
		src := src.([]byte)
		if len(src) != 64 {
			return fmt.Errorf("invalid token length: %d", len(src))
		}
		copy(b[:], src)
		return nil
	case string:
		src := src.(string)
		if len(src) != 64 {
			return fmt.Errorf("invalid token length: %d", len(src))
		}
		copy(b[:], src)
		return nil
	default:
		return fmt.Errorf("could not convert %T to Base64Token", src)
	}
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
	CaptchaID Base64Token
	Solution  CaptchaSolution
}

// Zeroes c on no captcha in request
// It is up to the caller to decide, if the returned error should or should not
// be ignored.
func (c *Captcha) FromRequest(r *http.Request) (err error) {
	c.CaptchaID, err = captchouli.ExtractID(r)
	if err != nil {
		return
	}
	c.Solution, err = captchouli.ExtractSolution(r)
	return
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
