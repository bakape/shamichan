package common

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/bakape/meguca/config"
)

// Extract host for setting cookies from request
func ExtractCookieHost(r *http.Request) string {
	host := r.URL.Hostname()

	// IP, not a domain name
	if net.ParseIP(host) != nil {
		return host
	}

	split := strings.Split(host, ".")
	l := len(split)
	switch {
	case l == 2:
		return "." + host
	case l < 2:
		return config.RootDomain()
	default:
		// Return the parent domain
		return fmt.Sprintf(".%s.%s", split[l-2], split[l-1])
	}
}
