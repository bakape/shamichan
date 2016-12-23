package auth

import (
	"bufio"
	"errors"
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
	ID       string    `gorethink:"id"`
	Password []byte    `gorethink:"password"`
	Sessions []Session `gorethink:"sessions"`
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

// Error during authenticating a captcha. These are not reported to the client,
// only logged.
type errCaptcha struct {
	error
}

func (e errCaptcha) Error() string {
	return "captcha: " + e.error.Error()
}

// BcryptCompare compares a bcrypt hash with a user-supplied string
func BcryptCompare(password string, hash []byte) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(password))
}

// AuthenticateCaptcha posts a request to the SolveMedia API to authenticate a
// captcha
func AuthenticateCaptcha(captcha common.Captcha, ip string) bool {
	conf := config.Get()

	// Captchas disabled or running tests. Can not use API, when testing
	if !conf.Captcha {
		return true
	}

	if captcha.Captcha == "" || captcha.CaptchaID == "" {
		return false
	}

	data := url.Values{
		"privatekey": {conf.CaptchaPrivateKey},
		"challenge":  {captcha.CaptchaID},
		"response":   {captcha.Captcha},
		"remoteip":   {ip},
	}
	res, err := http.PostForm("http://verify.solvemedia.com/papi/verify", data)
	if err != nil {
		printCaptchaError(err)
		return false
	}
	defer res.Body.Close()

	reader := bufio.NewReader(res.Body)
	status, err := reader.ReadString('\n')
	if err != nil {
		printCaptchaError(err)
		return false
	}
	if status[:len(status)-1] != "true" {
		reason, _ := reader.ReadString('\n')
		printCaptchaError(errors.New(reason[:len(reason)-1]))
		return false
	}

	return true
}

func printCaptchaError(err error) {
	log.Println(errCaptcha{err})
}

// HoldsPosition returns if the user holds a specific staff position
func HoldsPosition(board, userID, position string) bool {
	if staff := config.GetBoardConfigs(board).Staff; staff != nil {
		for _, o := range staff[position] {
			if o == userID {
				return true
			}
		}
	}
	return false
}

// FindPosition returns the first matching position of a user on a certain
// board. As a special case the admin user will always return "admin". If none
// found, returns empty string.
func FindPosition(board, userID string) string {
	if userID == "admin" {
		return userID
	}
	for pos, ids := range config.GetBoardConfigs(board).Staff {
		for _, id := range ids {
			if userID == id {
				return pos
			}
		}
	}
	return ""
}
