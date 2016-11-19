// Package auth determines and asserts client permissions to access and modify
// server resources.
package auth

import (
	"bufio"
	"encoding/base64"
	"errors"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
	"golang.org/x/crypto/bcrypt"
)

var (
	// IsReverseProxied specifies, if the server is deployed behind a reverse
	// proxy.
	IsReverseProxied bool

	// ReverseProxyIP specifies the IP of a non-localhost reverse proxy. Used
	// for filtering in XFF IP determination.
	ReverseProxyIP string
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

// Ident is used to verify a client's access and write permissions. Contains its
// IP and logged in user data, if any.
type Ident struct {
	UserID string
	IP     string
}

// Error during authenticating a captcha. These are not reported to the client,
// only logged.
type errCaptcha struct {
	error
}

func (e errCaptcha) Error() string {
	return "captcha: " + e.error.Error()
}

// LookUpIdent determine access rights of an IP
func LookUpIdent(req *http.Request) Ident {
	ident := Ident{
		IP: GetIP(req),
	}
	return ident
}

// IsBoard confirms the string is a valid board
func IsBoard(board string) bool {
	if board == "all" {
		return true
	}
	return IsNonMetaBoard(board)
}

// IsNonMetaBoard returns whether a valid board is a classic board and not
// some other path that emulates a board
func IsNonMetaBoard(b string) bool {
	return config.IsBoard(b)
}

// GetIP extracts the IP of a request, honouring reverse proxies, if set
func GetIP(req *http.Request) string {
	if IsReverseProxied {
		for _, h := range [...]string{"X-Forwarded-For", "X-Real-Ip"} {
			addresses := strings.Split(req.Header.Get(h), ",")

			// March from right to left until we get a public address.
			// That will be the address right before our reverse proxy.
			for i := len(addresses) - 1; i >= 0; i-- {
				// Header can contain padding spaces
				ip := strings.TrimSpace(addresses[i])

				// Filter the reverse proxy IPs
				switch {
				case ip == ReverseProxyIP:
				case !net.ParseIP(ip).IsGlobalUnicast():
				default:
					return ip
				}
			}
		}
	}
	return req.RemoteAddr
}

// RandomID generates a randomID of base64 characters of desired byte length
func RandomID(length int) (string, error) {
	buf := make([]byte, length)
	_, err := rand.Read(buf)
	return base64.RawStdEncoding.EncodeToString(buf), err
}

// BcryptHash generates a bcrypt hash from the passed string
func BcryptHash(password string, rounds int) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), rounds)
}

// BcryptCompare compares a bcrypt hash with a user-supplied string
func BcryptCompare(password string, hash []byte) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(password))
}

// AuthenticateCaptcha posts a request to the SolveMedia API to authenticate a
// captcha
func AuthenticateCaptcha(captcha types.Captcha, ip string) bool {
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
