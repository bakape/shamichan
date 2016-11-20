package server

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
)

const samplePassword = "123456"

var sampleLoginCreds = loginCredentials{
	UserID:  "user1",
	Session: genSession(),
}

func writeSampleUser(t *testing.T) {
	hash, err := auth.BcryptHash(samplePassword, 3)
	if err != nil {
		t.Fatal(err)
	}
	assertInsert(t, "accounts", auth.User{
		ID:       sampleLoginCreds.UserID,
		Password: hash,
		Sessions: []auth.Session{
			{
				Token:   sampleLoginCreds.Session,
				Expires: time.Now().Add(time.Hour),
			},
		},
	})
}

func genSession() string {
	return genString(common.LenSession)
}

func TestIsLoggedIn(t *testing.T) {
	assertTableClear(t, "accounts")

	token := genSession()
	assertInsert(t, "accounts", []auth.User{
		{
			ID: "user1",
			Sessions: []auth.Session{
				{
					Token:   token,
					Expires: time.Now().Add(time.Hour),
				},
			},
		},
		{
			ID:       "user2",
			Sessions: []auth.Session{},
		},
	})

	cases := [...]struct {
		name, user, session string
		isValid             bool
	}{
		{"valid", "user1", token, true},
		{"invalid session", "user2", genSession(), false},
		{"not registered", "nope", genSession(), false},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec, req := newPair("/")
			isValid := isLoggedIn(rec, req, c.user, c.session)
			if isValid != c.isValid {
				LogUnexpected(t, c.isValid, isValid)
			}
			if !c.isValid {
				assertCode(t, rec, 403)
				assertBody(t, rec, "403 invalid login credentials\n")
			}
		})
	}
}

func TestNotLoggedIn(t *testing.T) {
	assertTableClear(t, "accounts")

	fns := [...]http.HandlerFunc{
		configureBoard, servePrivateBoardConfigs, servePrivateServerConfigs,
		changePassword,
	}

	for i := range fns {
		fn := fns[i]
		t.Run("", func(t *testing.T) {
			t.Parallel()

			rec, req := newJSONPair(t, "/", sampleLoginCreds)
			fn(rec, req)
			assertCode(t, rec, 403)
			assertBody(t, rec, "403 invalid login credentials\n")
		})
	}
}

func TestChangePassword(t *testing.T) {
	assertTableClear(t, "accounts")
	writeSampleUser(t)

	const new = "654321"

	cases := [...]struct {
		name, old, new string
		code           int
		err            error
	}{
		{
			name: "wrong password",
			old:  "1234567",
			new:  new,
			code: 403,
			err:  errInvalidCreds,
		},
		{
			name: "new password too long",
			old:  samplePassword,
			new:  genString(common.MaxLenPassword + 1),
			code: 400,
			err:  errInvalidPassword,
		},
		{
			name: "empty new password",
			old:  samplePassword,
			new:  "",
			code: 400,
			err:  errInvalidPassword,
		},
		{
			name: "correct password",
			old:  samplePassword,
			new:  new,
			code: 200,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			msg := passwordChangeRequest{
				loginCredentials: sampleLoginCreds,
				Old:              c.old,
				New:              c.new,
			}
			rec, req := newJSONPair(t, "/admin/changePassword", msg)

			router.ServeHTTP(rec, req)

			assertCode(t, rec, c.code)
			if c.err != nil {
				assertBody(t, rec, fmt.Sprintf("%d %s\n", c.code, c.err))
			}
		})
	}

	// Assert new hash matches new password
	hash, err := db.GetLoginHash(sampleLoginCreds.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.BcryptCompare(new, hash); err != nil {
		t.Fatal(err)
	}
}
