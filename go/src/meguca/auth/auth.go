// Package auth determines and asserts client permissions to access and modify
// server resources.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"meguca/config"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

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

// Ban holdsan entry of an IP being banned from a board
type Ban struct {
	IP, Board string
}

// BanRecord stores information about a specific ban
type BanRecord struct {
	Ban
	ForPost    uint64
	Reason, By string
	Expires    time.Time
}

// IsBoard confirms the string is a valid board
func IsBoard(board string) bool {
	return board == "all" || IsNonMetaBoard(board)
}

// IsNonMetaBoard returns whether a valid board is a classic board and not
// some other path that emulates a board
func IsNonMetaBoard(b string) bool {
	return config.IsBoard(b)
}

// GetIP extracts the IP of a request, honouring reverse proxies, if set
func GetIP(r *http.Request) (string, error) {
	ip := getIP(r)
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP: %s", ip)
	}
	return ip, nil
}

func getIP(req *http.Request) string {
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
func IsBanned(board, ip string) (banned bool) {
	bansMu.RLock()
	defer bansMu.RUnlock()
	global := bans["all"]
	ips := bans[board]

	if global != nil && global[ip] {
		return true
	}
	if ips != nil && ips[ip] {
		return true
	}
	return false
}

// GetBannedLevels is like IsBanned, but returns, if the IP is banned globally
// or only from the specific board.
func GetBannedLevels(board, ip string) (globally, locally bool) {
	bansMu.RLock()
	defer bansMu.RUnlock()
	global := bans["all"]
	ips := bans[board]
	return global != nil && global[ip], ips != nil && ips[ip]
}

// SetBans replaces the ban cache with the new set
func SetBans(b ...Ban) {
	new := map[string]map[string]bool{}
	for _, b := range b {
		board, ok := new[b.Board]
		if !ok {
			board = map[string]bool{}
			new[b.Board] = board
		}
		board[b.IP] = true
	}

	bansMu.Lock()
	bans = new
	bansMu.Unlock()
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
