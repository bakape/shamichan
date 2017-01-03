// Package auth determines and asserts client permissions to access and modify
// server resources.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

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

	// board: IP: IsBanned
	bans   = map[string]map[string]bool{}
	bansMu sync.RWMutex
)

// BanRecord stores information about a specific ban
type BanRecord struct {
	ID      [2]string `gorethink:"id"`
	Reason  string    `gorethink:"reason"`
	By      string    `gorethink:"by"`
	Expires time.Time `gorethink:"expires"`
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
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr // No port in address
	}
	return ip
}

// IsBanned returns if the IP is banned on the target board
func IsBanned(board, ip string) bool {
	bansMu.RLock()
	ips, ok := bans[board]
	bansMu.RUnlock()

	if !ok {
		return false
	}
	return ips[ip]
}

// AddBan adds an IP to the banned cache of a board
func AddBan(board, ip string) {
	bansMu.Lock()
	defer bansMu.Unlock()

	ips, ok := bans[board]
	if !ok {
		ips = map[string]bool{}
		bans[board] = ips
	}
	ips[ip] = true
}

// RemoveBan removes an IP's ban from a specific board from the ban cache
func RemoveBan(board, ip string) {
	bansMu.Lock()
	defer bansMu.Unlock()

	ips, ok := bans[board]
	if !ok {
		return
	}
	if len(ips) == 1 {
		delete(bans, board)
	} else {
		delete(ips, ip)
	}
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
