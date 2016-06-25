package websockets

import (
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
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
	c.Assert(cl.loggedIn, Equals, true)
	_, msg, err := wcl.ReadMessage()
	c.Assert(err, IsNil)
	c.Assert(string(msg[:23]), Equals, `34{"code":0,"session":"`)
}

func (*DB) TestRegisterAlreadyLoggedIn(c *C) {
	cl := &Client{
		loggedIn: true,
	}
	c.Assert(register(nil, cl), Equals, errAlreadyLoggedIn)
	c.Assert(login(nil, cl), Equals, errAlreadyLoggedIn)
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
	c.Assert(fn(marshalJSON(req, c), cl), IsNil)
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
