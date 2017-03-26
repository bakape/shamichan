package auth

import (
	"encoding/json"
	"log"
	"meguca/config"
	"net/http"
	"net/url"
)

// AuthenticateCaptcha posts a request to the SolveMedia API to authenticate a
// captcha
func AuthenticateCaptcha(captcha string) bool {
	conf := config.Get()

	// Captchas disabled or running tests. Can not use API, when testing
	if !conf.Captcha {
		return true
	}
	if captcha == "" {
		return false
	}

	res, err := http.PostForm(
		"https://www.google.com/recaptcha/api/siteverify",
		url.Values{
			"secret":   {conf.CaptchaPrivateKey},
			"response": {captcha},
		},
	)
	if err != nil {
		printCaptchaError(err)
		return false
	}
	defer res.Body.Close()

	var data struct {
		Success bool
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		printCaptchaError(err)
		return false
	}
	return data.Success
}

func printCaptchaError(err error) {
	log.Printf("captcha: %s\n", err)
}
