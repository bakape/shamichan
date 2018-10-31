package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"meguca/config"
	"meguca/test"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCaptchaService(t *testing.T) {
	config.Set(config.Configs{
		CaptchaTags: config.Defaults.CaptchaTags,
		OverrideCaptchaTags: map[string][]string{
			"a": {"sakura_kyouko"},
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
		"captchouli-id": {b64},
	}
	for _, i := range std.Solution {
		q.Set(fmt.Sprintf("captchouli-%d", i), "on")
	}
	r := httptest.NewRequest("POST", "/", strings.NewReader(q.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	var c Captcha
	c.FromRequest(r)
	test.AssertDeepEquals(t, c, std)

	src, err := json.Marshal(std)
	if err != nil {
		return
	}
	err = json.Unmarshal(src, &c)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertDeepEquals(t, c, std)
}
