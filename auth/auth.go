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
	"github.com/jackc/pgx/pgtype"
)

// GetIP extracts the IP of a request, honouring reverse proxies, if set
func GetIP(r *http.Request) (ip net.IP, err error) {
	s := getIP(r)
	ip = net.ParseIP(s)
	if ip == nil {
		err = fmt.Errorf("invalid IP: %s", s)
		return
	}
	return
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
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(buf), nil
}

// 64 byte token that JSON/text en/decodes to a raw URL-safe encoding base64
// string
type AuthToken [64]byte

func (b AuthToken) MarshalText() ([]byte, error) {
	buf := make([]byte, 86)
	base64.RawURLEncoding.Encode(buf[:], b[:])
	return buf, nil
}

func (b *AuthToken) UnmarshalText(buf []byte) error {
	if len(buf) != 86 {
		return ErrInvalidToken
	}

	n, err := base64.RawURLEncoding.Decode(b[:], buf)
	if n != 64 || err != nil {
		return ErrInvalidToken
	}
	return nil
}

// Implement pgtype.Encoder
func (b AuthToken) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) (
	[]byte, error,
) {
	return append(buf, b[:]...), nil
}

// Create new AuthToken populated by cryptographically secure random data
func NewAuthToken() (b AuthToken, err error) {
	n, err := rand.Read(b[:])
	if err == nil && n != 64 {
		err = fmt.Errorf("auth: not enough data read: %d", n)
	}
	return
}
