package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"golang.org/x/crypto/bcrypt"
)

// User contains ID, password hash and board-related data of a registered user
// account
type User struct {
	ID       string
	Password []byte
}

// Session contains the token and expiry time of a single authenticated login
// session
type Session struct {
	Token   string    `gorethink:"token"`
	Expires time.Time `gorethink:"expires"`
}

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
func AuthenticateCaptcha(captcha common.Captcha) bool {
	conf := config.Get()

	// Captchas disabled or running tests. Can not use API, when testing
	if !conf.Captcha {
		return true
	}

	if captcha.Captcha == "" {
		return false
	}

	res, err := http.PostForm(
		"https://www.google.com/recaptcha/api/siteverify",
		url.Values{
			"secret":   {conf.CaptchaPrivateKey},
			"response": {captcha.Captcha},
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
