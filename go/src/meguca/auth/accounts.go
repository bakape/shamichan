package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"meguca/config"
	"golang.org/x/crypto/bcrypt"
)

// SessionCreds is embed in every request that needs logged in authentication
type SessionCreds struct {
	UserID, Session string
}

type captchaValidationResponse struct {
	Success bool
}

// BcryptCompare compares a bcrypt hash with a user-supplied string
func BcryptCompare(password string, hash []byte) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(password))
}

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

	var data captchaValidationResponse
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		printCaptchaError(err)
		return false
	}
	return data.Success
}

func printCaptchaError(err error) {
	log.Printf("captcha: %s\n", err)
}
