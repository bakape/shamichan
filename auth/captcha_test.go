package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/bakape/captchouli"
	"github.com/Chiiruno/meguca/config"
	"github.com/Chiiruno/meguca/test"
)

func TestCaptchaService(t *testing.T) {
	// Skip to avoid massive booru fetches on DB population
	test.SkipInCI(t)

	config.Set(config.Configs{
		CaptchaTags: config.Defaults.CaptchaTags,
		OverrideCaptchaTags: map[string]string{
			"a": "sakura_kyouko",
		},
		Public: config.Public{
			Captcha: true,
		},
	})

	err := LoadCaptchaServices()
	if err != nil {
		t.Fatal(err)
	}

	if CaptchaService("a") == nil {
		t.Fatal()
	}
	_, err = CreateTestCaptcha()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDecodeCaptcha(t *testing.T) {
	std := Captcha{
		Solution: []byte{1, 2, 3},
	}
	_, err := rand.Read(std.CaptchaID[:])
	if err != nil {
		t.Fatal(err)
	}
	b64 := base64.StdEncoding.EncodeToString(std.CaptchaID[:])

	q := url.Values{
		captchouli.IDKey: {b64},
	}
	for _, i := range std.Solution {
		q.Set(fmt.Sprintf("captchouli-%d", i), "on")
	}
	r := httptest.NewRequest("POST", "/", strings.NewReader(q.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	var c Captcha
	c.FromRequest(r)
	test.AssertEquals(t, c, std)

	src, err := json.Marshal(std)
	if err != nil {
		return
	}
	err = json.Unmarshal(src, &c)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, c, std)
}

func TestNewToken(t *testing.T) {
	t.Parallel()

	_, err := NewBase64Token()
	if err != nil {
		t.Fatal(err)
	}
}
