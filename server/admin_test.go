package server

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
	r "github.com/dancannon/gorethink"
)

var sampleLoginCredentials = loginCredentials{
	UserID:  "user1",
	Session: "token1",
}

func TestIsLoggedIn(t *testing.T) {
	assertTableClear(t, "accounts")
	assertInsert(t, "accounts", []auth.User{
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
	})

	cases := [...]struct {
		name, user, session string
		isValid             bool
	}{
		{"valid", "user1", "token1", true},
		{"invalid session", "user2", "token2", false},
		{"not registered", "nope", "token3", false},
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
				assertBody(t, rec, "403 Invalid login credentials\n")
			}
		})
	}
}

func TestNotLoggedIn(t *testing.T) {
	assertTableClear(t, "accounts")

	fns := [...]http.HandlerFunc{configureBoard, servePrivateBoardConfigs}

	for i := range fns {
		fn := fns[i]
		t.Run("", func(t *testing.T) {
			t.Parallel()

			rec, req := newJSONPair(t, "/", sampleLoginCredentials)
			fn(rec, req)
			assertCode(t, rec, 403)
			assertBody(t, rec, "403 Invalid login credentials\n")
		})
	}
}

func newJSONPair(t *testing.T, url string, data interface{}) (
	*httptest.ResponseRecorder, *http.Request,
) {
	body := encodeBody(t, data)
	return httptest.NewRecorder(), httptest.NewRequest("POST", url, body)
}

func encodeBody(t *testing.T, data interface{}) io.Reader {
	return bytes.NewReader(marshalJSON(t, data))
}

func TestNotBoardOwner(t *testing.T) {
	assertTableClear(t, "accounts")
	writeSampleUser(t)

	fns := [...]http.HandlerFunc{configureBoard, servePrivateBoardConfigs}

	for i := range fns {
		fn := fns[i]
		t.Run("", func(t *testing.T) {
			t.Parallel()

			rec, req := newJSONPair(t, "/", sampleLoginCredentials)
			fn(rec, req)
			assertCode(t, rec, 403)
			assertBody(t, rec, "403 Not board owner\n")
		})
	}
}

func writeSampleUser(t *testing.T) {
	assertInsert(t, "accounts", auth.User{
		ID: "user1",
		Sessions: []auth.Session{
			{
				Token:   "token1",
				Expires: time.Now().Add(time.Hour),
			},
		},
	})
}

func TestServePrivateBoardConfigs(t *testing.T) {
	assertTableClear(t, "accounts")
	writeSampleUser(t)

	conf := config.BoardConfigs{
		ID:        "a",
		Eightball: []string{"a", "b", "c"},
		BoardPublic: config.BoardPublic{
			Banners: []string{},
		},
		Staff: map[string][]string{
			"owners": {"user1"},
		},
	}
	config.SetBoardConfigs(conf)

	rec, req := newJSONPair(t, "/admin/boardConfig", boardConfigRequest{
		ID:               "a",
		loginCredentials: sampleLoginCredentials,
	})
	router.ServeHTTP(rec, req)
	assertBody(t, rec, string(marshalJSON(t, conf)))
}

func TestBoardConfiguration(t *testing.T) {
	assertTableClear(t, "accounts", "boards")

	const (
		id    = "user1"
		board = "a"
	)
	staff := map[string][]string{
		"owners": {id},
	}
	conf := config.BoardConfigs{
		ID: board,
		BoardPublic: config.BoardPublic{
			PostParseConfigs: config.PostParseConfigs{
				ForcedAnon: true,
			},
			Banners: []string{},
			Spoiler: "default.jpg",
		},
		Eightball: []string{},
		Staff:     staff,
	}
	init := config.BoardConfigs{
		ID: board,
		BoardPublic: config.BoardPublic{
			Banners: []string{},
		},
		Eightball: []string{},
		Staff:     staff,
	}
	assertInsert(t, "boards", init)

	writeSampleUser(t)

	data := boardConfigSettingRequest{
		loginCredentials: sampleLoginCredentials,
		BoardConfigs:     conf,
	}
	rec, req := newJSONPair(t, "/admin/configureBoard", data)
	router.ServeHTTP(rec, req)

	var res config.BoardConfigs
	if err := db.One(r.Table("boards").Get(board), &res); err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, res, conf)
}

func TestValidateConfigs(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name string
		config.BoardConfigs
		err error
	}{
		{
			"all is well",
			config.BoardConfigs{},
			nil,
		},
		{
			"too many eightball answers",
			config.BoardConfigs{
				Eightball: make([]string, maxEigthballLen+1),
			},
			errTooManyAnswers,
		},
		{
			"compound eightball length to big",
			config.BoardConfigs{
				Eightball: []string{genString(maxEigthballLen + 1)},
			},
			errEightballTooLong,
		},
		{
			"notice too long",
			config.BoardConfigs{
				BoardPublic: config.BoardPublic{
					Notice: genString(maxNoticeLen + 1),
				},
			},
			errNoticeTooLong,
		},
		{
			"rules too long",
			config.BoardConfigs{
				BoardPublic: config.BoardPublic{
					Rules: genString(maxRulesLen + 1),
				},
			},
			errRulesTooLong,
		},
		{
			"title too long",
			config.BoardConfigs{
				BoardPublic: config.BoardPublic{
					Title: genString(maxTitleLen + 1),
				},
			},
			errTitleTooLong,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			if b := validateConfigs(rec, c.BoardConfigs); b != (c.err == nil) {
				t.Fatal("unexpected result")
			}
			if c.err != nil {
				assertCode(t, rec, 400)
				assertBody(t, rec, fmt.Sprintf("400 %s\n", c.err))
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
