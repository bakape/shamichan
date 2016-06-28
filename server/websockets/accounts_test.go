package websockets

import (
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
	"golang.org/x/crypto/bcrypt"
	. "gopkg.in/check.v1"
)

var (
	wrongCredentialsResopnse = []byte(`34{"code":2,"session":""}`)
)

func (*DB) TestRegistrationStringValidations(c *C) {
	r21, err := util.RandomID(21)
	c.Assert(err, IsNil)
	r31, err := util.RandomID(31)
	c.Assert(err, IsNil)

	samples := []struct {
		id, password string
		code         loginResponseCode
	}{
		{"12", "123456", idTooShort},
		{r21, "123456", idTooLong},
		{"123", "12345", passwordTooShort},
		{"123", r31, passwordTooLong},
	}

	for _, s := range samples {
		code, err := handleRegistration(s.id, s.password)
		c.Assert(err, IsNil)
		c.Assert(code, Equals, s.code)
	}
}

func (*DB) TestRegistration(c *C) {
	req := loginRequest{
		ID:       "123",
		Password: "123456",
	}

	// Valid registration
	assertValidLogin(req, register, c)

	// User name taken
	assertHandlerResponse(req, register, []byte(`34{"code":1,"session":""}`), c)
}

func assertValidLogin(req interface{}, fn handler, c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	c.Assert(fn(marshalJSON(req, c), cl), IsNil)
	c.Assert(cl.isLoggedIn(), Equals, true)
	c.Assert(cl.userID, Not(Equals), "")
	_, msg, err := wcl.ReadMessage()
	c.Assert(err, IsNil)
	c.Assert(string(msg[:23]), Equals, `34{"code":0,"session":"`)
}

func (*DB) TestAlreadyLoggedIn(c *C) {
	cl := &Client{
		sessionToken: "foo",
	}
	for _, fn := range [...]handler{register, login, authenticateSession} {
		c.Assert(fn(nil, cl), Equals, errAlreadyLoggedIn)
	}
}

func (*DB) TestNotLoggedIn(c *C) {
	cl := new(Client)
	for _, fn := range [...]handler{logOut, logOutAll, changePassword} {
		c.Assert(fn(nil, cl), Equals, errNotLoggedIn)
	}
}

func (*DB) TestNoUserRegistered(c *C) {
	req := loginRequest{
		ID:       "123",
		Password: "1233456",
	}
	assertHandlerResponse(req, login, wrongCredentialsResopnse, c)
}

func assertHandlerResponse(req interface{}, fn handler, msg []byte, c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	data := marshalJSON(req, c)
	c.Assert(fn(data, cl), IsNil)
	assertMessage(wcl, msg, c)
}

func (*DB) TestLogin(c *C) {
	const (
		id       = "123"
		password = "123456"
	)
	hash, err := bcrypt.GenerateFromPassword([]byte(id+password), 10)
	c.Assert(err, IsNil)
	c.Assert(db.RegisterAccount(id, hash), IsNil)
	req := loginRequest{
		ID:       id,
		Password: password,
	}

	// Valid login
	assertValidLogin(req, login, c)

	// Wrong password
	req.Password += "1"
	assertHandlerResponse(req, login, wrongCredentialsResopnse, c)
}

func (*DB) TestAuthenticateNonExistantUser(c *C) {
	req := authenticationRequest{
		ID: "123",
	}
	assertHandlerResponse(req, authenticateSession, []byte("35false"), c)
}

func (*DB) TestAuthenticateInvalidSession(c *C) {
	const id = "123"
	req := authenticationRequest{
		ID: "123",
	}
	c.Assert(db.RegisterAccount(id, []byte("bar")), IsNil)

	assertHandlerResponse(req, authenticateSession, []byte("35false"), c)
}

func (*DB) TestAuthentication(c *C) {
	const (
		id      = "123"
		session = "foo"
	)
	user := auth.User{
		ID: id,
		Sessions: []auth.Session{
			{
				Token:   session,
				Expires: time.Now().Add(30 * 24 * time.Hour),
			},
		},
	}
	c.Assert(db.Write(r.Table("accounts").Insert(user)), IsNil)

	req := authenticationRequest{
		ID:      id,
		Session: session,
	}

	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	data := marshalJSON(req, c)
	c.Assert(authenticateSession(data, cl), IsNil)
	c.Assert(cl.sessionToken, Equals, session)
	c.Assert(cl.userID, Equals, id)
	assertMessage(wcl, []byte("35true"), c)
}

func (*DB) TestLogOut(c *C) {
	const id = "123"
	sessions := []auth.Session{
		{Token: "foo"},
		{Token: "bar"},
	}
	user := auth.User{
		ID:       id,
		Sessions: sessions,
	}
	c.Assert(db.Write(r.Table("accounts").Insert(user)), IsNil)

	assertLogout(id, logOut, c)

	// Assert database user document
	var res []auth.Session
	c.Assert(db.All(db.GetAccount(id).Field("sessions"), &res), IsNil)
	res[0].Expires = time.Time{}
	c.Assert(res, DeepEquals, []auth.Session{sessions[1]})
}

func assertLogout(id string, fn handler, c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()

	cl.userID = id
	cl.sessionToken = "foo"

	c.Assert(fn(nil, cl), IsNil)
	assertMessage(wcl, []byte("36true"), c)
	c.Assert(cl.userID, Equals, "")
	c.Assert(cl.sessionToken, Equals, "")
}

func (*DB) TestLogOutAll(c *C) {
	const id = "123"
	sessions := []auth.Session{
		{Token: "foo"},
		{Token: "bar"},
	}
	user := auth.User{
		ID:       id,
		Sessions: sessions,
		Password: []byte{1, 2, 3},
		Rigths:   []auth.Right{},
	}
	c.Assert(db.Write(r.Table("accounts").Insert(user)), IsNil)

	assertLogout(id, logOutAll, c)

	// Assert database user document
	var res auth.User
	c.Assert(db.One(db.GetAccount(id), &res), IsNil)
	user.Sessions = []auth.Session{}
	c.Assert(res, DeepEquals, user)
}

func (*DB) TestChangePassword(c *C) {
	const (
		id  = "123"
		old = "123456"
		new = "654321"
	)
	hash, err := bcrypt.GenerateFromPassword([]byte(old), 10)
	c.Assert(err, IsNil)
	c.Assert(db.RegisterAccount(id, hash), IsNil)

	// Wrong password
	req := passwordChangeRequest{
		Old: "1234567",
		New: new,
	}
	assertLoggedInResponse(req, changePassword, id, []byte("38false"), c)

	// Correct password
	req = passwordChangeRequest{
		Old: old,
		New: new,
	}
	assertLoggedInResponse(req, changePassword, id, []byte("38true"), c)

	// Assert new hash was generated from the old
	hash, err = db.GetLoginHash(id)
	c.Assert(err, IsNil)
	c.Assert(bcrypt.CompareHashAndPassword(hash, []byte(new)), IsNil)
}

func assertLoggedInResponse(
	req interface{},
	fn handler,
	id string,
	msg []byte,
	c *C,
) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	cl.sessionToken = "foo"
	cl.userID = id
	data := marshalJSON(req, c)
	c.Assert(fn(data, cl), IsNil)
	assertMessage(wcl, msg, c)
}
