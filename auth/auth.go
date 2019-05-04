// Package auth determines and asserts client permissions to access and modify
// server resources.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/bakape/meguca/config"

	"golang.org/x/crypto/bcrypt"
)

// IsBoard confirms the string is a valid board
func IsBoard(board string) bool {
	return board == "all" || IsNonMetaBoard(board)
}

// IsNonMetaBoard returns whether a valid board is a classic board and not
// some other path that emulates a board
func IsNonMetaBoard(b string) bool {
	return b != "all" && config.IsBoard(b)
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
	var ip string
	if config.Server.Server.ReverseProxied {
		h := req.Header.Get("X-Forwarded-For")
		if h != "" {
			if i := strings.LastIndexByte(h, ','); i != -1 {
				h = h[i+1:]
			}

			ip = strings.TrimSpace(h) // Header can contain padding spaces
		}
	}
	if ip == "" {
		ip = req.RemoteAddr
	}

	split, _, err := net.SplitHostPort(ip)
	if err != nil {
		return ip // No port in address
	}
	return split
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
