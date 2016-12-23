// Package auth determines and asserts client permissions to access and modify
// server resources.
package auth

import (
	"encoding/base64"
	"math/rand"
	"net"
	"net/http"
	"strings"

	"github.com/bakape/meguca/config"
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

// Ident is used to verify a client's access and write permissions. Contains its
// IP and logged in user data, if any.
type Ident struct {
	UserID string
	IP     string
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
