package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

var sampleLoginCredentials = loginCredentials{
	UserID:  "user1",
	Session: "token1",
}

func (d *DB) TestIsLoggedIn(c *C) {
	users := []auth.User{
		{
			ID: "user1",
			Sessions: []auth.Session{
				{
					Token:   "token1",
					Expires: time.Now().Add(time.Hour),
				},
			},
		},
		{
			ID:       "user2",
			Sessions: []auth.Session{},
		},
	}
	c.Assert(db.Write(r.Table("accounts").Insert(users)), IsNil)

	samples := [...]struct {
		user, session string
		isValid       bool
	}{
		{"user1", "token1", true},
		{"user2", "token2", false},
		{"notAUser", "token3", false},
	}

	for _, s := range samples {
		rec, req := newPair(c, "/")
		isValid := isLoggedIn(rec, req, s.user, s.session)
		c.Assert(isValid, Equals, s.isValid)
		if !s.isValid {
			assertCode(rec, 403, c)
			assertBody(rec, "403 Invalid login credentials\n", c)
		}
	}
}

func (*DB) TestNotLoggedIn(c *C) {
	fns := [...]http.HandlerFunc{configureBoard, servePrivateBoardConfigs}

	for _, fn := range fns {
		rec, req := newJSONPair(c, "/", sampleLoginCredentials)
		fn(rec, req)
		assertCode(rec, 403, c)
		assertBody(rec, "403 Invalid login credentials\n", c)
	}
}

func newJSONPair(c *C, url string, data interface{}) (
	*httptest.ResponseRecorder, *http.Request,
) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", url, encodeBody(c, data))
	return rec, req
}

func encodeBody(c *C, data interface{}) io.Reader {
	return bytes.NewReader(marshalJSON(c, data))
}

func marshalJSON(c *C, data interface{}) []byte {
	buf, err := json.Marshal(data)
	c.Assert(err, IsNil)
	return buf
}

func (*DB) TestNotBoardOwner(c *C) {
	writeSampleUser(c)

	fns := [...]http.HandlerFunc{configureBoard, servePrivateBoardConfigs}

	for _, fn := range fns {
		rec, req := newJSONPair(c, "/", sampleLoginCredentials)
		fn(rec, req)
		assertCode(rec, 403, c)
		assertBody(rec, "403 Not board owner\n", c)
	}
}

func writeSampleUser(c *C) {
	user := auth.User{
		ID: "user1",
		Sessions: []auth.Session{
			{
				Token:   "token1",
				Expires: time.Now().Add(time.Hour),
			},
		},
	}
	c.Assert(db.Write(r.Table("accounts").Insert(user)), IsNil)
}

func (d *DB) TestServePrivateBoardConfigs(c *C) {
	writeSampleUser(c)

	conf := config.BoardConfigs{
		ID:        "a",
		Eightball: []string{"a", "b", "c"},
		Banners:   []string{},
		Staff: map[string][]string{
			"owners": {"user1"},
		},
	}
	c.Assert(db.Write(r.Table("boards").Insert(conf)), IsNil)

	rec, req := newJSONPair(c, "/admin/boardConfig", boardConfigRequest{
		ID:               "a",
		loginCredentials: sampleLoginCredentials,
	})
	d.r.ServeHTTP(rec, req)
	assertBody(rec, string(marshalJSON(c, conf)), c)
}

func (d *DB) TestBoardConfiguration(c *C) {
	const (
		id    = "user1"
		board = "a"
	)
	staff := map[string][]string{
		"owners": {id},
	}
	conf := config.BoardConfigs{
		ID: board,
		PostParseConfigs: config.PostParseConfigs{
			ForcedAnon: true,
		},
		Spoiler:   "default.jpg",
		Eightball: []string{},
		Banners:   []string{},
		Staff:     staff,
	}
	init := config.BoardConfigs{
		ID:        board,
		Eightball: []string{},
		Banners:   []string{},
		Staff:     staff,
	}
	c.Assert(db.Write(r.Table("boards").Insert(init)), IsNil)

	writeSampleUser(c)

	data := boardConfigSettingRequest{
		loginCredentials: sampleLoginCredentials,
		BoardConfigs:     conf,
	}
	rec, req := newJSONPair(c, "/admin/configureBoard", data)
	d.r.ServeHTTP(rec, req)

	var res config.BoardConfigs
	c.Assert(db.One(db.GetBoardConfig(board), &res), IsNil)
	c.Assert(res, DeepEquals, conf)
}

func (*WebServer) TestValidateConfigs(c *C) {
	samples := [...]struct {
		config.BoardConfigs
		err error
	}{
		{}, // All is well
		{
			BoardConfigs: config.BoardConfigs{
				Eightball: make([]string, maxEigthballLen+1),
			},
			err: errTooManyAnswers,
		},
		{
			BoardConfigs: config.BoardConfigs{
				Eightball: []string{genString(maxEigthballLen + 1)},
			},
			err: errEightballTooLong,
		},
		{
			BoardConfigs: config.BoardConfigs{
				Notice: genString(maxNoticeLen + 1),
			},
			err: errNoticeTooLong,
		},
		{
			BoardConfigs: config.BoardConfigs{
				Rules: genString(maxRulesLen + 1),
			},
			err: errRulesTooLong,
		},
		{
			BoardConfigs: config.BoardConfigs{
				Title: genString(maxTitleLen + 1),
			},
			err: errTitleTooLong,
		},
	}

	for _, s := range samples {
		rec := httptest.NewRecorder()
		c.Assert(validateConfigs(rec, s.BoardConfigs), Equals, s.err == nil)
		if s.err != nil {
			assertCode(rec, 400, c)
			assertBody(rec, fmt.Sprintf("400 %s\n", s.err), c)
		}
	}
}

// Generate a test string of suplied length
func genString(len int) string {
	return strings.Repeat("a", len)
}
