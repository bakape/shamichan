package auth

import (
	"net/http/httptest"
	"testing"

	"meguca/config"
	. "meguca/test"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	config.Set(config.Configs{})
}

func TestIsBoard(t *testing.T) {
	config.Clear()
	_, err := config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
	})
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name, in string
		isBoard  bool
	}{
		{"exits", "a", true},
		{"doesn't exist", "b", false},
		{"/all/ board", "all", true},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			if IsBoard(c.in) != c.isBoard {
				t.Fatal("unexpected result")
			}
		})
	}
}

func TestGetIP(t *testing.T) {
	const (
		ip             = "207.178.71.93"
		reverseProxyIP = "162.30.251.246"
	)
	IsReverseProxied = true
	ReverseProxyIP = reverseProxyIP

	cases := [...]struct {
		name, xff, out string
	}{
		{"valid XFF", "10.121.169.19", "10.121.169.19"},
		{"no XFF", "", ip},
		{"invalid XFF", "notip, nope", ip},
		{
			"hosted on localhost",
			"105.124.243.122, 10.168.239.157, 127.0.0.1, ::1",
			"10.168.239.157",
		},
		{
			"behind reverse proxy",
			"105.124.243.122," + reverseProxyIP,
			"105.124.243.122",
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = ip
			if c.xff != "" {
				req.Header.Set("X-Forwarded-For", c.xff)
			}
			if i := GetIP(req); i != c.out {
				LogUnexpected(t, c.out, ip)
			}
		})
	}
}

func TestBcryptHash(t *testing.T) {
	t.Parallel()

	const (
		password = "123456"
	)
	hash, err := BcryptHash(password, 8)
	if err != nil {
		t.Fatal(err)
	}

	// Mismatch
	err = BcryptCompare(password+"1", hash)
	if err != bcrypt.ErrMismatchedHashAndPassword {
		UnexpectedError(t, err)
	}

	// Correct
	if err := BcryptCompare(password, hash); err != nil {
		t.Fatal(err)
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
