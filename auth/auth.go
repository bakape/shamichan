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
