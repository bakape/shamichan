package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/Chiiruno/meguca/common"
	"github.com/Chiiruno/meguca/config"
	. "github.com/Chiiruno/meguca/test"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	config.Set(config.Configs{})
	common.IsTest = true
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
