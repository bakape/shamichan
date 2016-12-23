package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	return GenString(common.LenSession)
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
	hash, err := db.GetLoginHash(sampleLoginCreds.UserID)
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
			rec, req := newJSONPair(t, "/admin/register", loginRequest{
				loginCreds: loginCreds{
					ID:       c.id,
					Password: c.password,
				},
			})
			router.ServeHTTP(rec, req)

			assertError(t, rec, c.code, c.err)
			if c.err == nil {
				assertLogin(t, c.id, rec.Body.String())
			}
		})
	}
}

func assertLogin(t *testing.T, user, session string) {
	var contains bool
	q := db.GetAccount(user).Field("sessions").Field("token").Contains(session)
	if err := db.One(q, &contains); err != nil {
		t.Fatal(err)
	}
	if !contains {
		t.Fatal("session not created")
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

			rec, req := newJSONPair(t, "/admin/login", loginRequest{
				loginCreds: loginCreds{
					ID:       c.id,
					Password: c.password,
				},
			})
			router.ServeHTTP(rec, req)

			assertError(t, rec, c.code, c.err)
			if c.err == nil {
				assertLogin(t, c.id, rec.Body.String())
			}
		})
	}
}

func TestLogout(t *testing.T) {
	assertTableClear(t, "accounts")

	const id = "123"
	sessions := []auth.Session{
		{Token: genSession()},
		{Token: genSession()},
	}
	assertInsert(t, "accounts", auth.User{
		ID:       id,
		Sessions: sessions,
	})

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
			token: sessions[0].Token,
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

			// Assert database user document
			if c.err == nil {
				var res []auth.Session
				err := db.All(db.GetAccount(id).Field("sessions"), &res)
				if err != nil {
					t.Fatal(err)
				}
				res[0].Expires = time.Time{} // Normalize time
				AssertDeepEquals(t, res, []auth.Session{sessions[1]})
			}
		})
	}
}

func TestLogoutAll(t *testing.T) {
	assertTableClear(t, "accounts")

	const id = "123"
	sessions := []auth.Session{
		{Token: genSession()},
		{Token: genSession()},
	}
	user := auth.User{
		ID:       id,
		Sessions: sessions,
		Password: []byte{1, 2, 3},
	}
	assertInsert(t, "accounts", user)

	rec, req := newJSONPair(t, "/admin/logoutAll", auth.SessionCreds{
		UserID:  id,
		Session: sessions[0].Token,
	})
	router.ServeHTTP(rec, req)

	assertCode(t, rec, 200)

	var res auth.User
	if err := db.One(db.GetAccount(id), &res); err != nil {
		t.Fatal(err)
	}
	user.Sessions = []auth.Session{}
	AssertDeepEquals(t, res, user)
}
