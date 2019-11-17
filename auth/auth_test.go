package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
)

func init() {
	config.Set(config.Configs{})
	common.IsTest = true
}

func TestGetIP(t *testing.T) {
	const (
		ip             = "207.178.71.93"
		reverseProxyIP = "162.30.251.246"
	)
	config.Server.Server.ReverseProxied = true

	cases := [...]struct {
		name, xff, out string
	}{
		{
			name: "valid XFF",
			xff:  "10.121.169.19",
			out:  "10.121.169.19",
		},
		{
			name: "no XFF",
			out:  ip,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			// t.Parallel()

			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = ip
			if c.xff != "" {
				req.Header.Set("X-Forwarded-For", c.xff)
			}

			res, err := GetIP(req)
			if err != nil {
				t.Fatal(err)
			}
			AssertEquals(t, res, c.out)
		})
	}
}

func TestRandomID(t *testing.T) {
	t.Parallel()

	hash, err := RandomID(32)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(hash); l != 43 {
		t.Fatalf("unexpected hash string length: %d", l)
	}
}
