package websockets

import (
	"github.com/bakape/meguca/util"
	. "gopkg.in/check.v1"
)

func (*DB) TestRegistrationStringValidations(c *C) {
	samples := []struct {
		id, password string
		code         accountResponse
	}{
		{"12", "123456", idTooShort},
		{util.RandomID(21), "123456", idTooLong},
		{"123", "12345", passwordTooShort},
		{"123", util.RandomID(31), passwordTooLong},
	}

	for _, s := range samples {
		code, err := handleRegistration(s.id, s.password)
		c.Assert(err, IsNil)
		c.Assert(code, Equals, s.code)
	}
}

func (*DB) TestRegistration(c *C) {
	req := registrationRequest{
		ID:       "123",
		Password: "123456",
	}
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	data := marshalJSON(req, c)

	// Valid registration
	sv.Add(1)
	c.Assert(register(data, cl), IsNil)
	msg, err := encodeMessage(messageLogin, loginSuccess)
	c.Assert(err, IsNil)
	c.Assert(cl.loggedIn, Equals, true)
	assertMessage(wcl, msg, sv, c)
	sv.Wait()

	// User name taken
	cl, wcl = sv.NewClient()
	sv.Add(1)
	c.Assert(register(data, cl), IsNil)
	msg, err = encodeMessage(messageLogin, userNameTaken)
	c.Assert(err, IsNil)
	c.Assert(cl.loggedIn, Equals, false)
	assertMessage(wcl, msg, sv, c)
	sv.Wait()
}

func (*DB) TestAlreadyLoggedIn(c *C) {
	cl := &Client{
		loggedIn: true,
	}
	c.Assert(register(nil, cl), ErrorMatches, "already logged in")
}
