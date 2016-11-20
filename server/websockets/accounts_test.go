package websockets

import (
	"bytes"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
)

const (
	wrongCredentialsResponse = `34{"code":2,"session":""}`
)

func TestRegistrationValidations(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, id, password string
		code               loginResponseCode
	}{
		{"id too short", "", "123456", idTooShort},
		{"id too long", genString(21), "123456", idTooLong},
		{"password too short", "123", "", passwordTooShort},
		{"password too long", "123", genString(31), passwordTooLong},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			req := loginRequest{
				ID:       c.id,
				Password: c.password,
			}
			cl := &Client{
				Ident: auth.Ident{
					IP: "::1",
				},
			}
			code, err := handleRegistration(req, cl)
			if err != nil {
				t.Fatal(err)
			}
			if code != c.code {
				LogUnexpected(t, c.code, code)
			}
		})
	}
}

func genString(len int) string {
	var buf bytes.Buffer
	for i := 0; i < len; i++ {
		buf.WriteRune(rune(rand.Intn(128)))
	}
	return buf.String()
}

func TestRegistration(t *testing.T) {
	assertTableClear(t, "accounts")

	req := loginRequest{
		ID:       "123",
		Password: "123456",
	}

	// Valid registration
	assertValidLogin(t, req, register, MessageRegister)

	// User name taken
	assertHandlerResponse(t, req, register, `33{"code":1,"session":""}`)
}

func assertValidLogin(
	t *testing.T,
	req interface{},
	fn handler,
	typ MessageType,
) {
	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	if err := fn(marshalJSON(t, req), cl); err != nil {
		t.Fatal(err)
	}
	if !cl.isLoggedIn() {
		t.Fatal("not logged in")
	}
	if cl.UserID == "" {
		t.Fatal("empty user ID")
	}

	_, msg, err := wcl.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	std := strconv.Itoa(int(typ)) + `{"code":0,"session":"`
	if s := string(msg[:23]); s != std {
		LogUnexpected(t, std, s)
	}
}

func assertHandlerResponse(
	t *testing.T,
	req interface{},
	fn handler,
	msg string,
) {
	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	if err := fn(marshalJSON(t, req), cl); err != nil {
		t.Fatal(err)
	}
	assertMessage(t, wcl, msg)
}

func TestAlreadyLoggedIn(t *testing.T) {
	t.Parallel()

	cl := &Client{
		sessionToken: "foo",
	}
	for _, fn := range [...]handler{register, login, authenticateSession} {
		if err := fn(nil, cl); err != errAlreadyLoggedIn {
			UnexpectedError(t, err)
		}
	}
}

func TestNotLoggedIn(t *testing.T) {
	t.Parallel()

	cl := new(Client)
	fns := [...]handler{logOut, logOutAll}
	for _, fn := range fns {
		if err := fn(nil, cl); err != errNotLoggedIn {
			UnexpectedError(t, err)
		}
	}
}

func TestNoUserRegistered(t *testing.T) {
	assertTableClear(t, "accounts")

	req := loginRequest{
		ID:       "123",
		Password: "1233456",
	}
	assertHandlerResponse(t, req, login, wrongCredentialsResponse)
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
	req := loginRequest{
		ID:       id,
		Password: password,
	}

	t.Run("valid login", func(t *testing.T) {
		t.Parallel()
		assertValidLogin(t, req, login, MessageLogin)
	})
	t.Run("wrong password", func(t *testing.T) {
		t.Parallel()
		r := req
		r.Password += "1"
		assertHandlerResponse(t, r, login, wrongCredentialsResponse)
	})
}

func TestAuthentication(t *testing.T) {
	assertTableClear(t, "accounts")

	const (
		id      = "123"
		session = "foo"
	)
	assertInsert(t, "accounts", auth.User{
		ID: id,
		Sessions: []auth.Session{
			{
				Token:   session,
				Expires: time.Now().Add(30 * 24 * time.Hour),
			},
		},
	})

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	data := marshalJSON(t, authenticationRequest{
		ID:      id,
		Session: session,
	})

	if err := authenticateSession(data, cl); err != nil {
		t.Fatal(err)
	}
	if cl.sessionToken != session {
		LogUnexpected(t, session, cl.sessionToken)
	}
	if cl.UserID != id {
		LogUnexpected(t, id, cl.UserID)
	}
	assertMessage(t, wcl, "35true")

	t.Run("invalid session", func(t *testing.T) {
		t.Parallel()

		const id = "abcdefg"
		req := authenticationRequest{
			ID: id,
		}
		if err := db.RegisterAccount(id, []byte("bar")); err != nil {
			t.Fatal(err)
		}

		assertHandlerResponse(t, req, authenticateSession, "35false")
	})

	t.Run("nonexistent user", func(t *testing.T) {
		t.Parallel()

		req := authenticationRequest{
			ID: "dASDFSDSSD",
		}
		assertHandlerResponse(t, req, authenticateSession, "35false")
	})
}

func TestLogOut(t *testing.T) {
	assertTableClear(t, "accounts")

	const id = "123"
	sessions := []auth.Session{
		{Token: "foo"},
		{Token: "bar"},
	}
	assertInsert(t, "accounts", auth.User{
		ID:       id,
		Sessions: sessions,
	})

	assertLogout(t, id, logOut)

	// Assert database user document
	var res []auth.Session
	if err := db.All(db.GetAccount(id).Field("sessions"), &res); err != nil {
		t.Fatal(err)
	}
	res[0].Expires = time.Time{}
	std := []auth.Session{sessions[1]}
	AssertDeepEquals(t, res, std)
}

func assertLogout(t *testing.T, id string, fn handler) {
	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	cl.UserID = id
	cl.sessionToken = "foo"

	if err := fn(nil, cl); err != nil {
		t.Fatal(err)
	}

	assertMessage(t, wcl, "36true")
	if cl.UserID != "" {
		t.Fatal("user id retained")
	}
	if cl.sessionToken != "" {
		t.Fatal("user token retained")
	}
}

func TestLogOutAll(t *testing.T) {
	assertTableClear(t, "accounts")

	const id = "123"
	sessions := []auth.Session{
		{Token: "foo"},
		{Token: "bar"},
	}
	user := auth.User{
		ID:       id,
		Sessions: sessions,
		Password: []byte{1, 2, 3},
	}
	assertInsert(t, "accounts", user)

	assertLogout(t, id, logOutAll)

	// Assert database user document
	var res auth.User
	if err := db.One(db.GetAccount(id), &res); err != nil {
		t.Fatal(err)
	}
	user.Sessions = []auth.Session{}
	AssertDeepEquals(t, res, user)
}

func assertLoggedInResponse(
	t *testing.T,
	req interface{},
	fn handler,
	id, msg string,
) {
	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	cl.sessionToken = "foo"
	cl.UserID = id

	if err := fn(marshalJSON(t, req), cl); err != nil {
		t.Fatal(err)
	}
	assertMessage(t, wcl, msg)
}
