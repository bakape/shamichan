package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
)

const samplePassword = "123456"

var sampleLoginCreds = auth.SessionCreds{
	UserID:  "user1",
	Session: genSession(),
}

func writeSampleUser(t *testing.T) {
	hash, err := auth.BcryptHash(samplePassword, 3)
	if err != nil {
		t.Fatal(err)
	}
	err = db.RegisterAccount(sampleLoginCreds.UserID, hash)
	if err != nil {
		t.Fatal(err)
	}
	err = db.WriteLoginSession(
		sampleLoginCreds.UserID,
		sampleLoginCreds.Session,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func genSession() string {
	return GenString(common.LenSession)
}

func TestIsLoggedIn(t *testing.T) {
	assertTableClear(t, "accounts")

	hash, err := auth.BcryptHash(samplePassword, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RegisterAccount("user1", hash); err != nil {
		t.Fatal(err)
	}
	if err := db.RegisterAccount("user2", hash); err != nil {
		t.Fatal(err)
	}

	token := genSession()
	if err := db.WriteLoginSession("user1", token); err != nil {
		t.Fatal(err)
	}

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
				assertError(t, rec, 403, common.ErrInvalidCreds)
			}
		})
	}
}

func assertError(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	code int,
	err error,
) {
	assertCode(t, rec, code)
	if err != nil {
		assertBody(t, rec, fmt.Sprintf("%d %s\n", code, err))
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
			assertError(t, rec, 403, common.ErrInvalidCreds)
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
			err:  common.ErrInvalidCreds,
		},
		{
			name: "new password too long",
			old:  samplePassword,
			new:  GenString(common.MaxLenPassword + 1),
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
				SessionCreds: sampleLoginCreds,
				Old:          c.old,
				New:          c.new,
			}
			rec, req := newJSONPair(t, "/admin/changePassword", msg)

			router.ServeHTTP(rec, req)

			assertError(t, rec, c.code, c.err)
		})
	}

	// Assert new hash matches new password
	hash, err := db.GetPassword(sampleLoginCreds.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.BcryptCompare(new, hash); err != nil {
		t.Fatal(err)
	}
}

func TestRegistrationValidations(t *testing.T) {
	assertTableClear(t, "accounts")

	cases := [...]struct {
		name, id, password string
		code               int
		err                error
	}{
		{
			name:     "no ID",
			id:       "",
			password: "123456",
			code:     400,
			err:      errInvalidUserID,
		},
		{
			name:     "id too long",
			id:       GenString(common.MaxLenUserID + 1),
			password: "123456",
			code:     400,
			err:      errInvalidUserID,
		},
		{
			name:     "no password",
			id:       "123",
			password: "",
			code:     400,
			err:      errInvalidPassword,
		},
		{
			name:     "password too long",
			id:       "123",
			password: GenString(common.MaxLenPassword + 1),
			code:     400,
			err:      errInvalidPassword,
		},
		{
			name:     "valid",
			id:       "123",
			password: "456",
			code:     200,
		},
		{
			name:     "id already taken",
			id:       "123",
			password: "456",
			code:     400,
			err:      errUserIDTaken,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			rec, req := newJSONPair(t, "/admin/register", loginCreds{
				ID:       c.id,
				Password: c.password,
			})
			router.ServeHTTP(rec, req)

			assertError(t, rec, c.code, c.err)
			if c.err == nil {
				assertLogin(t, c.id, rec.Body.String(), true)
			}
		})
	}
}

func assertLogin(t *testing.T, user, session string, loggedIn bool) {
	is, err := db.IsLoggedIn(user, session)
	switch {
	case err != nil:
		t.Fatal(err)
	case is != loggedIn:
		t.Fatalf("unexpected session status: %t", is)
	}
}

func TestLogin(t *testing.T) {
	assertTableClear(t, "accounts")

	const (
		id       = "123"
		password = "123456"
	)
	hash, err := auth.BcryptHash(password, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RegisterAccount(id, hash); err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name, id, password string
		code               int
		err                error
	}{
		{
			name:     "invalid ID",
			id:       id + "1",
			password: password,
			code:     403,
			err:      common.ErrInvalidCreds,
		},
		{
			name:     "invalid password",
			id:       id,
			password: password + "1",
			code:     403,
			err:      common.ErrInvalidCreds,
		},
		{
			name:     "valid",
			id:       id,
			password: password,
			code:     200,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec, req := newJSONPair(t, "/admin/login", loginCreds{
				ID:       c.id,
				Password: c.password,
			})
			router.ServeHTTP(rec, req)

			assertError(t, rec, c.code, c.err)
			if c.err == nil {
				assertLogin(t, c.id, rec.Body.String(), true)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	assertTableClear(t, "accounts")
	id, tokens := writeSampleSessions(t)

	cases := [...]struct {
		name, token string
		code        int
		err         error
	}{
		{
			name:  "not logged in",
			token: genSession(),
			code:  403,
			err:   common.ErrInvalidCreds,
		},
		{
			name:  "valid",
			token: tokens[0],
			code:  200,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec, req := newJSONPair(t, "/admin/logout", auth.SessionCreds{
				UserID:  id,
				Session: c.token,
			})
			router.ServeHTTP(rec, req)

			assertError(t, rec, c.code, c.err)

			if c.err == nil {
				assertLogin(t, id, tokens[0], false)
				assertLogin(t, id, tokens[1], true)
			}
		})
	}
}

func writeSampleSessions(t *testing.T) (string, [2]string) {
	const id = "123"
	tokens := [2]string{genSession(), genSession()}
	hash, err := auth.BcryptHash("foo", 3)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.RegisterAccount(id, hash); err != nil {
		t.Fatal(err)
	}
	for _, token := range tokens {
		if err := db.WriteLoginSession(id, token); err != nil {
			t.Fatal(err)
		}
	}

	return id, tokens
}

func TestLogoutAll(t *testing.T) {
	assertTableClear(t, "accounts")
	id, tokens := writeSampleSessions(t)

	rec, req := newJSONPair(t, "/admin/logoutAll", auth.SessionCreds{
		UserID:  id,
		Session: tokens[0],
	})
	router.ServeHTTP(rec, req)

	assertCode(t, rec, 200)
	for _, tok := range tokens {
		assertLogin(t, id, tok, false)
	}
}
